package llm

import (
	"context"
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

	// LLMClient 用于生成结构化摘要的 LLM 客户端
	// 当为 nil 时回退到启发式截断压缩
	llmClient LLMClient

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
// llmClient 用于 LLM 驱动的结构化摘要，传 nil 则回退到启发式截断
func DefaultContextCompressor(llmClient LLMClient) *ContextCompressor {
	return &ContextCompressor{
		ContextWindowSize: 128000, // gpt-4o default
		CompressThreshold: 0.75,
		RecentKeepRounds:  3,
		llmClient:         llmClient,
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
func (c *ContextCompressor) StartAsyncCompress(ctx context.Context, historySnapshot []Message) {
	c.mu.Lock()
	if c.compressing {
		c.mu.Unlock()
		return // 已有压缩任务在运行，不重复启动
	}
	c.compressing = true
	c.compressedReady = false
	c.mu.Unlock()

	go func() {
		// 后台压缩使用独立的 context，避免原始请求超时取消压缩
		bgCtx := context.Background()
		_ = ctx // 原始 context 仅作为参考，后台任务用独立 context
		compressed := c.CompressHistory(bgCtx, historySnapshot)

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
// 3. 对较早的轮次，两级压缩：先 Prune 工具输出，再 LLM 结构化摘要
// 4. D&D 领域感知：查询类工具激进压缩，动作类工具保守压缩
// 返回压缩后的消息列表
func (c *ContextCompressor) CompressHistory(ctx context.Context, messages []Message) []Message {
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
	compressed := c.compressOldMessages(ctx, oldMessages)

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

// compressOldMessages 压缩旧消息（两级压缩策略）
// Level 1: Prune - 修剪大型工具输出，减少发送给摘要 LLM 的 token 量
// Level 2: LLM Summarize - 用 LLM 生成结构化摘要，失败时回退到启发式摘要
func (c *ContextCompressor) compressOldMessages(ctx context.Context, messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}

	// Level 1: Prune - 修剪大型工具输出
	pruned := c.pruneOldMessages(messages)

	// Level 2: LLM Summarize
	if c.llmClient != nil {
		summary, err := c.summarizeWithLLM(ctx, pruned)
		if err == nil && summary != "" {
			summaryContent := fmt.Sprintf("[\u5386\u53f2\u4e0a\u4e0b\u6587\u6458\u8981 - \u4ee5\u4e0b\u662f\u4e4b\u524d\u5bf9\u8bdd\u7684\u7ed3\u6784\u5316\u538b\u7f29\u7248\u672c]\n%s", summary)
			return []Message{
				NewSystemMessage(summaryContent),
			}
		}
		// LLM 摘要失败，回退到启发式摘要
	}

	// 回退：启发式摘要（当 LLMClient 为 nil 或 LLM 调用失败时）
	return c.compressOldMessagesHeuristic(pruned)
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

// summarizeSegment 将一个消息段压缩为简短摘要（启发式回退方案）
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

// --- Level 1: Prune - 修剪大型工具输出 ---

// pruneOldMessages 修剪旧消息中的大型工具输出
// - 查询类工具结果替换为元信息标注
// - 大型工具输出（>500 token）截断保留前 100 token
// - 工具调用参数中的大 JSON（>200 token）截断
func (c *ContextCompressor) pruneOldMessages(messages []Message) []Message {
	pruned := make([]Message, len(messages))

	// 收集当前批次中所有工具名，用于将 tool result 匹配到工具类型
	toolCallNames := make(map[string]string) // toolCallID -> toolName
	for _, msg := range messages {
		if msg.Role == RoleAssistant {
			for _, tc := range msg.ToolCalls {
				toolCallNames[tc.ID] = tc.Name
			}
		}
	}

	for i, msg := range messages {
		switch msg.Role {
		case RoleTool:
			// 确定对应的工具名
			toolName := toolCallNames[msg.Name]
			if isQueryTool(toolName) {
				// 查询类工具：激进压缩，只保留元信息
				pruned[i] = Message{
					Role:    msg.Role,
					Content: fmt.Sprintf("[\u67e5\u8be2\u7ed3\u679c\u5df2\u7701\u7565\uff0c\u5de5\u5177: %s\uff0c\u53ef\u901a\u8fc7 GameSummary \u6062\u590d]", toolName),
					Name:    msg.Name,
				}
			} else {
				// 动作类工具：如果输出过大则截断
				prunedMsg := msg
				tokens := estimateStringTokens(msg.Content)
				if tokens > 500 {
					// 保留前 ~100 token 对应的字符数（约 400 字符）
					keepChars := 400
					if len(msg.Content) > keepChars {
						prunedMsg.Content = msg.Content[:keepChars] + fmt.Sprintf("\n...[\u5df2\u622a\u65ad\uff0c\u539f\u59cb\u7ea6 %d tokens]", tokens)
					}
				}
				pruned[i] = prunedMsg
			}

		case RoleAssistant:
			// 修剪工具调用参数中的大 JSON
			prunedMsg := msg
			if len(msg.ToolCalls) > 0 {
				newToolCalls := make([]ToolCall, len(msg.ToolCalls))
				for j, tc := range msg.ToolCalls {
					newTC := tc
					// 检查每个参数值的大小
					for k, v := range tc.Arguments {
						vStr := fmt.Sprintf("%v", v)
						if estimateStringTokens(vStr) > 200 {
							if newTC.Arguments == nil {
								newTC.Arguments = make(map[string]any)
								for kk, vv := range tc.Arguments {
									newTC.Arguments[kk] = vv
								}
							}
							if len(vStr) > 200 {
								newTC.Arguments[k] = vStr[:200] + "...[\u5df2\u622a\u65ad]"
							}
						}
					}
					newToolCalls[j] = newTC
				}
				prunedMsg.ToolCalls = newToolCalls
			}
			pruned[i] = prunedMsg

		default:
			pruned[i] = msg
		}
	}

	return pruned
}

// --- Level 2: LLM 结构化摘要 ---

// compactionSystemPrompt D&D 专属结构化摘要 prompt
const compactionSystemPrompt = `你是一个D&D游戏会话的上下文压缩器。请将以下对话历史压缩为结构化摘要，用于让DM（地下城主）AI继续游戏。

必须包含以下部分（如果存在）：
1. 【情节进展】当前故事线和关键剧情事件
2. 【玩家行动】玩家做出的重要决策和行动
3. 【战斗/遭遇】进行中或已完成的战斗，关键结果（伤害、击杀、状态效果）
4. 【NPC交互】与NPC的重要对话和关系变化
5. 【物品/状态变更】获得/失去的物品、状态变化、位置移动
6. 【待处理事项】进行中但未完成的任务或悬而未决的情况
7. 【关键ID引用】涉及的重要实体ID（角色、场景、物品等）

要求：
- 保留所有实体ID（格式如 01HXXXXXX）
- 保留数值结果（骰子结果、HP变化等）
- 用简洁的条目式表达，不要叙事性文字
- 如果某个部分没有相关内容，跳过该部分`

// summarizeWithLLM 使用 LLM 生成结构化摘要
func (c *ContextCompressor) summarizeWithLLM(ctx context.Context, messages []Message) (string, error) {
	// 将消息格式化为可读文本
	userPrompt := c.buildSummarizationPrompt(messages)

	req := &CompletionRequest{
		Messages: []Message{
			NewSystemMessage(compactionSystemPrompt),
			NewUserMessage(userPrompt),
		},
		Temperature: 0, // 确保摘要确定性
		MaxTokens:   2000,
	}

	resp, err := c.llmClient.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM summarization failed: %w", err)
	}

	if resp.Content == "" {
		return "", fmt.Errorf("LLM returned empty summary")
	}

	return resp.Content, nil
}

// buildSummarizationPrompt 将消息列表格式化为可读文本，作为摘要 LLM 的用户输入
func (c *ContextCompressor) buildSummarizationPrompt(messages []Message) string {
	var sb strings.Builder
	sb.WriteString("请压缩以下对话历史。这些消息将被摘要替换，摘要将是继续游戏的唯一上下文，请保留所有关键信息。\n\n---\n")

	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			sb.WriteString(fmt.Sprintf("\n[玩家]: %s\n", msg.Content))

		case RoleAssistant:
			if msg.Content != "" {
				sb.WriteString(fmt.Sprintf("\n[DM]: %s\n", msg.Content))
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					// 简化参数显示
					var argParts []string
					for k, v := range tc.Arguments {
						argParts = append(argParts, fmt.Sprintf("%s=%v", k, v))
					}
					sb.WriteString(fmt.Sprintf("[DM 调用工具]: %s(%s)\n", tc.Name, strings.Join(argParts, ", ")))
				}
			}

		case RoleTool:
			sb.WriteString(fmt.Sprintf("[工具结果]: %s\n", msg.Content))

		case RoleSystem:
			sb.WriteString(fmt.Sprintf("[系统]: %s\n", msg.Content))
		}
	}

	sb.WriteString("\n---\n请按照指定格式生成结构化摘要。")
	return sb.String()
}

// --- 启发式摘要回退方案 ---

// compressOldMessagesHeuristic 启发式摘要（当 LLMClient 为 nil 或 LLM 调用失败时的回退方案）
func (c *ContextCompressor) compressOldMessagesHeuristic(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}

	segments := c.segmentMessages(messages)

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

	summaryContent := fmt.Sprintf("[\u5386\u53f2\u4e0a\u4e0b\u6587\u6458\u8981 - \u4ee5\u4e0b\u662f\u4e4b\u524d\u5bf9\u8bdd\u7684\u538b\u7f29\u7248\u672c]\n%s", strings.Join(summaryParts, "\n"))

	return []Message{
		NewSystemMessage(summaryContent),
	}
}
