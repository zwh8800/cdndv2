package llm

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

// ContextCompressor 上下文压缩器
// 当对话历史的估算Token数接近上下文窗口上限时，自动压缩历史消息
// 支持异步后台压缩，不阻塞用户正常交互流程
type ContextCompressor struct {
	// ContextWindowSize 模型上下文窗口大小（token数）
	ContextWindowSize int

	// CompressThreshold 压缩触发阈值（0-1），默认0.75表示75%
	CompressThreshold float64

	// RecentKeepRounds 保留最近的完整对话轮次数（不被压缩），默认3
	// 一个"轮次"定义为从一条 user message 开始，到下一条 user message（不含）为止
	RecentKeepRounds int

	// Token 估算校准
	lastActualTokens    int     // 上次 API 返回的实际 prompt token 数
	lastEstimatedTokens int     // 上次估算值
	calibrationRatio    float64 // 校准系数，默认 1.3（偏保守）

	// 异步压缩状态
	mu               sync.Mutex
	compressing      bool      // 是否正在后台压缩
	compressedResult []Message // 后台压缩完成的结果（待替换）
	compressedReady  bool      // 压缩结果是否就绪
}

// DefaultContextCompressor 创建默认配置的压缩器
func DefaultContextCompressor() *ContextCompressor {
	return &ContextCompressor{
		ContextWindowSize: 128000, // gpt-4o default
		CompressThreshold: 0.75,
		RecentKeepRounds:  3,
		calibrationRatio:  1.3, // 偏保守，后续通过实际 token 反馈动态调整
	}
}

// --- Token 估算 ---

// EstimateTokens 估算消息列表的Token数
// 采用启发式估算并乘以校准系数
func (c *ContextCompressor) EstimateTokens(messages []Message) int {
	raw := 0
	for _, msg := range messages {
		raw += estimateMessageTokens(msg)
	}
	ratio := c.calibrationRatio
	if ratio <= 0 {
		ratio = 1.3
	}
	return int(float64(raw) * ratio)
}

// estimateRawTokens 估算原始Token数（不含校准系数）
func (c *ContextCompressor) estimateRawTokens(messages []Message) int {
	raw := 0
	for _, msg := range messages {
		raw += estimateMessageTokens(msg)
	}
	return raw
}

// estimateMessageTokens 估算单条消息的Token数
func estimateMessageTokens(msg Message) int {
	tokens := 4 // 每条消息的固定开销（role, separators等）

	// 估算内容Token数
	tokens += estimateStringTokens(msg.Content)

	// 估算Tool调用Token数
	for _, tc := range msg.ToolCalls {
		tokens += 3 // tool call结构开销
		tokens += estimateStringTokens(tc.Name)
		tokens += estimateStringTokens(tc.ID)
		for k, v := range tc.Arguments {
			tokens += estimateStringTokens(k)
			tokens += estimateStringTokens(fmt.Sprintf("%v", v))
		}
	}

	// 估算Tool结果Token数
	for _, tr := range msg.ToolResults {
		tokens += estimateStringTokens(tr.Content)
		tokens += estimateStringTokens(tr.ToolCallID)
	}

	// Name 字段（用于 tool message 的 tool_call_id）
	if msg.Name != "" {
		tokens += estimateStringTokens(msg.Name)
	}

	return tokens
}

// estimateStringTokens 估算字符串的Token数
func estimateStringTokens(s string) int {
	if s == "" {
		return 0
	}

	// 统计中文和非中文字符
	chineseChars := 0
	totalChars := utf8.RuneCountInString(s)

	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF || // CJK Unified Ideographs
			r >= 0x3400 && r <= 0x4DBF || // CJK Extension A
			r >= 0x3000 && r <= 0x303F || // CJK Symbols and Punctuation
			r >= 0xFF00 && r <= 0xFFEF { // Fullwidth Forms
			chineseChars++
		}
	}

	nonChineseChars := totalChars - chineseChars

	// 中文约2字符/token，英文约4字符/token
	chineseTokens := (chineseChars + 1) / 2
	nonChineseTokens := (nonChineseChars + 3) / 4

	return chineseTokens + nonChineseTokens
}

// --- Token 校准 ---

// CalibrateWithActualUsage 用实际 API 返回的 token 数校准估算模型
// 使用指数移动平均（EMA）平滑更新校准系数
func (c *ContextCompressor) CalibrateWithActualUsage(estimatedTokens, actualPromptTokens int) {
	if actualPromptTokens <= 0 || estimatedTokens <= 0 {
		return
	}
	c.lastActualTokens = actualPromptTokens
	c.lastEstimatedTokens = estimatedTokens
	// 指数移动平均更新校准系数
	// 注意：这里用的 estimatedTokens 是已乘过旧 ratio 的值，需要还原为 raw
	rawEstimated := c.estimateRawTokens(nil) // 不能直接用，需要用传入的值反推
	if c.calibrationRatio > 0 {
		rawEstimated = int(float64(estimatedTokens) / c.calibrationRatio)
	}
	if rawEstimated <= 0 {
		rawEstimated = estimatedTokens
	}
	newRatio := float64(actualPromptTokens) / float64(rawEstimated)
	c.calibrationRatio = c.calibrationRatio*0.7 + newRatio*0.3
}

// --- 压缩触发判断 ---

// NeedsCompression 检查消息历史是否需要压缩
func (c *ContextCompressor) NeedsCompression(messages []Message) bool {
	estimated := c.EstimateTokens(messages)
	threshold := int(float64(c.ContextWindowSize) * c.CompressThreshold)
	return estimated > threshold
}

// --- 异步压缩 ---

// IsCompressing 返回是否有后台压缩任务正在运行
func (c *ContextCompressor) IsCompressing() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.compressing
}

// StartAsyncCompress 启动后台异步压缩
// 传入当前历史的快照（copy），在后台执行压缩
// 压缩完成后结果存入 compressedResult，等待下次调用 ApplyCompressedIfReady 时替换
func (c *ContextCompressor) StartAsyncCompress(historySnapshot []Message) {
	c.mu.Lock()
	if c.compressing {
		c.mu.Unlock()
		return // 已有压缩任务在运行，不重复启动
	}
	c.compressing = true
	c.compressedReady = false
	c.mu.Unlock()

	go func() {
		compressed := c.CompressHistory(historySnapshot)

		c.mu.Lock()
		c.compressedResult = compressed
		c.compressedReady = true
		c.compressing = false
		c.mu.Unlock()
	}()
}

// ApplyCompressedIfReady 如果后台压缩已完成，返回压缩后的历史；否则返回 nil
func (c *ContextCompressor) ApplyCompressedIfReady() []Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.compressedReady {
		return nil
	}

	result := c.compressedResult
	c.compressedResult = nil
	c.compressedReady = false
	return result
}

// --- 核心压缩逻辑 ---

// CompressHistory 压缩对话历史
// 策略：
// 1. 按"对话轮次"划分历史（每个轮次从 user message 开始）
// 2. 保留最近 RecentKeepRounds 个完整轮次不动
// 3. 对较早的轮次，按消息段分类压缩（用户输入、工具调用序列、助手响应）
// 4. D&D 领域感知：查询类工具激进压缩，动作类工具保守压缩
// 返回压缩后的消息列表
func (c *ContextCompressor) CompressHistory(messages []Message) []Message {
	rounds := c.identifyRounds(messages)

	keepRounds := c.RecentKeepRounds
	if keepRounds <= 0 {
		keepRounds = 3
	}

	if len(rounds) <= keepRounds {
		return messages
	}

	// 分割为旧轮次和最近轮次
	splitRoundIdx := len(rounds) - keepRounds
	oldRounds := rounds[:splitRoundIdx]
	recentRounds := rounds[splitRoundIdx:]

	// 收集旧轮次的所有消息并压缩
	var oldMessages []Message
	for _, r := range oldRounds {
		oldMessages = append(oldMessages, r.messages...)
	}

	// 收集最近轮次的所有消息（保留原样）
	var recentMessages []Message
	for _, r := range recentRounds {
		recentMessages = append(recentMessages, r.messages...)
	}

	// 压缩旧消息
	compressed := c.compressOldMessages(oldMessages)

	// 合并压缩后的消息和最近消息
	result := make([]Message, 0, len(compressed)+len(recentMessages))
	result = append(result, compressed...)
	result = append(result, recentMessages...)

	return result
}

// --- 轮次识别 ---

// conversationRound 对话轮次（从一条 user message 到下一条 user message 之前的所有消息）
type conversationRound struct {
	messages []Message
}

// identifyRounds 将消息列表划分为"对话轮次"
// 一个轮次定义为：从一条 user message 开始，到下一条 user message（不含）为止
// 如果开头没有 user message，则前面的消息归入第一个轮次
func (c *ContextCompressor) identifyRounds(messages []Message) []conversationRound {
	if len(messages) == 0 {
		return nil
	}

	var rounds []conversationRound
	var current []Message

	for _, msg := range messages {
		if msg.Role == RoleUser && len(current) > 0 {
			// 遇到新的 user message，结束当前轮次
			rounds = append(rounds, conversationRound{messages: current})
			current = nil
		}
		current = append(current, msg)
	}

	// 最后一个轮次
	if len(current) > 0 {
		rounds = append(rounds, conversationRound{messages: current})
	}

	return rounds
}

// --- 消息压缩 ---

// compressOldMessages 压缩旧消息
func (c *ContextCompressor) compressOldMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}

	// 将消息分组为"对话段"
	segments := c.segmentMessages(messages)

	// 压缩每个段为简短摘要
	var summaryParts []string
	for _, seg := range segments {
		summary := c.summarizeSegment(seg)
		if summary != "" {
			summaryParts = append(summaryParts, summary)
		}
	}

	if len(summaryParts) == 0 {
		return nil
	}

	// 生成一条压缩摘要消息（使用 system message，避免 LLM 误解为玩家输入）
	summaryContent := fmt.Sprintf("[历史上下文摘要 - 以下是之前对话的压缩版本]\n%s", strings.Join(summaryParts, "\n"))

	return []Message{
		NewSystemMessage(summaryContent),
	}
}

// messageSegment 消息段（一组相关的消息）
type messageSegment struct {
	messages []Message
	segType  segmentType
}

type segmentType int

const (
	segTypeUserInput    segmentType = iota // 用户输入
	segTypeToolSequence                    // 工具调用序列（assistant tool_call + tool results）
	segTypeAssistant                       // 纯 assistant 文本响应
)

// segmentMessages 将消息分组为对话段
func (c *ContextCompressor) segmentMessages(messages []Message) []messageSegment {
	var segments []messageSegment
	i := 0

	for i < len(messages) {
		msg := messages[i]

		switch msg.Role {
		case RoleUser:
			segments = append(segments, messageSegment{
				messages: []Message{msg},
				segType:  segTypeUserInput,
			})
			i++

		case RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// 收集 assistant(tool_calls) + 后续的 tool result 消息
				seg := messageSegment{
					messages: []Message{msg},
					segType:  segTypeToolSequence,
				}
				i++
				// 收集所有紧随的 tool 消息
				for i < len(messages) && messages[i].Role == RoleTool {
					seg.messages = append(seg.messages, messages[i])
					i++
				}
				segments = append(segments, seg)
			} else {
				segments = append(segments, messageSegment{
					messages: []Message{msg},
					segType:  segTypeAssistant,
				})
				i++
			}

		case RoleTool:
			// 孤立的 tool 消息（理论上不应该出现），归入工具序列
			seg := messageSegment{
				messages: []Message{msg},
				segType:  segTypeToolSequence,
			}
			i++
			for i < len(messages) && messages[i].Role == RoleTool {
				seg.messages = append(seg.messages, messages[i])
				i++
			}
			segments = append(segments, seg)

		default:
			i++
		}
	}

	return segments
}

// summarizeSegment 将一个消息段压缩为简短摘要
func (c *ContextCompressor) summarizeSegment(seg messageSegment) string {
	switch seg.segType {
	case segTypeUserInput:
		content := seg.messages[0].Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		return fmt.Sprintf("- 玩家: %s", content)

	case segTypeToolSequence:
		return c.summarizeToolSequence(seg.messages)

	case segTypeAssistant:
		content := seg.messages[0].Content
		if content == "" {
			return ""
		}
		if len(content) > 300 {
			content = content[:300] + "..."
		}
		return fmt.Sprintf("- DM: %s", content)

	default:
		return ""
	}
}

// --- D&D 领域感知的工具分类压缩 ---

// queryToolPrefixes 查询类工具前缀（结果可从 GameSummary 恢复，可激进压缩）
var queryToolPrefixes = []string{"get_", "list_", "query_"}

// isQueryTool 判断是否为查询类工具（结果可从 CollectSummary 恢复）
func isQueryTool(name string) bool {
	for _, prefix := range queryToolPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// summarizeToolSequence 压缩工具调用序列
// D&D 领域感知：查询类工具激进压缩，动作类工具保守压缩
func (c *ContextCompressor) summarizeToolSequence(messages []Message) string {
	if len(messages) == 0 {
		return ""
	}

	var queryTools []string  // 查询类工具名
	var actionTools []string // 动作类工具名
	var hasError bool
	var actionResults []string // 动作类工具的关键结果

	for _, msg := range messages {
		if msg.Role == RoleAssistant {
			for _, tc := range msg.ToolCalls {
				if isQueryTool(tc.Name) {
					queryTools = append(queryTools, tc.Name)
				} else {
					actionTools = append(actionTools, tc.Name)
				}
			}
		}
		if msg.Role == RoleTool {
			content := msg.Content
			if strings.HasPrefix(content, "Error:") || strings.HasPrefix(content, "错误:") {
				hasError = true
			}
			// 只为动作类工具保留结果（查询类工具结果可从 GameSummary 恢复）
			// 判断方法：按顺序对应 tool call 的位置，简化为全部检查是否含关键 ID
			if containsEntityID(content) || hasError {
				if len(content) > 150 {
					content = content[:150] + "..."
				}
				actionResults = append(actionResults, content)
			}
		}
	}

	if len(queryTools) == 0 && len(actionTools) == 0 {
		return ""
	}

	var parts []string

	// 查询类工具：激进压缩，只保留工具名
	if len(queryTools) > 0 {
		parts = append(parts, fmt.Sprintf("- 查询[%s]: 已获取数据", strings.Join(queryTools, ", ")))
	}

	// 动作类工具：保守压缩，保留关键结果
	if len(actionTools) > 0 {
		status := "成功"
		if hasError {
			status = "部分失败"
		}
		summary := fmt.Sprintf("- 执行[%s]: %s", strings.Join(actionTools, ", "), status)

		// 附加关键结果数据
		if len(actionResults) > 0 && len(actionResults) <= 3 {
			for _, r := range actionResults {
				summary += fmt.Sprintf("\n  结果: %s", r)
			}
		} else if len(actionResults) > 3 {
			summary += fmt.Sprintf("\n  结果(%d项): %s ... %s",
				len(actionResults), actionResults[0], actionResults[len(actionResults)-1])
		}

		parts = append(parts, summary)
	}

	return strings.Join(parts, "\n")
}

// containsEntityID 检查内容是否包含实体ID（ULID格式或关键字段）
func containsEntityID(content string) bool {
	// 检查是否包含 actor_id、scene_id 等关键字段
	keywords := []string{"actor_id", "scene_id", "game_id", "item_id", "npc_id"}
	lowerContent := strings.ToLower(content)
	for _, kw := range keywords {
		if strings.Contains(lowerContent, kw) {
			return true
		}
	}
	return false
}
