package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/zwh8800/cdndv2/game_engine/game_summary"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/prompt"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// MainAgent 主Agent(DM)实现
type MainAgent struct {
	registry  *tool.ToolRegistry
	llm       llm.LLMClient
	subAgents map[string]SubAgent
	logger    *zap.Logger
}

// NewMainAgent 创建主Agent
func NewMainAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient, subAgents map[string]SubAgent) *MainAgent {
	return &MainAgent{
		registry:  registry,
		llm:       llmClient,
		subAgents: subAgents,
		logger:    zap.NewNop(),
	}
}

// SetLogger 设置日志器
func (m *MainAgent) SetLogger(log *zap.Logger) {
	if log != nil {
		m.logger = log
	}
}

// getLogger 获取日志器
func (m *MainAgent) getLogger() *zap.Logger {
	if m.logger == nil {
		m.logger = zap.NewNop()
	}
	return m.logger
}

// Name 返回Agent名称
func (m *MainAgent) Name() string {
	return MainAgentName
}

// Description 返回Agent描述
func (m *MainAgent) Description() string {
	return "主Agent(DM)，负责意图理解、任务分解、叙事生成、玩家交互"
}

// SystemPrompt 返回系统提示词
func (m *MainAgent) SystemPrompt(ctx *AgentContext) string {
	// 加载提示词模板
	templateStr, err := prompt.LoadSystemPrompt("main_system.md")
	if err != nil {
		// 如果加载失败，使用默认提示词
		return m.defaultSystemPrompt(ctx)
	}

	// 准备模板数据
	data := m.prepareTemplateData(ctx)

	// 渲染模板
	rendered, err := prompt.RenderTemplate(templateStr, data)
	if err != nil {
		return m.defaultSystemPrompt(ctx)
	}

	return rendered
}

// Tools 返回Agent可用的Tools
func (m *MainAgent) Tools() []tool.Tool {
	return m.registry.GetAllTools()
}

// Execute 执行Agent逻辑
func (m *MainAgent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	log := m.getLogger()

	log.Debug("[MainAgent] Execute started",
		zap.String("userInput", req.UserInput),
		zap.Int("historyLength", len(req.Context.History)),
	)

	// 构建系统提示词
	systemPrompt := m.SystemPrompt(req.Context)
	log.Debug("[MainAgent] System prompt built",
		zap.Int("promptLength", len(systemPrompt)),
	)

	// 组装消息
	messages := m.buildMessages(systemPrompt, req)
	log.Debug("[MainAgent] Messages built",
		zap.Int("messageCount", len(messages)),
	)

	// 获取Tools定义
	tools := m.registry.GetAll()
	log.Debug("[MainAgent] Tools retrieved",
		zap.Int("toolCount", len(tools)),
	)

	// 打印发送给LLM的消息
	for i, msg := range messages {
		roleStr := string(msg.Role)
		content := msg.Content
		if len(content) > 300 {
			content = content[:300] + "..."
		}
		toolCallNames := make([]string, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			toolCallNames = append(toolCallNames, tc.Name)
		}
		log.Debug("[MainAgent] LLM request message",
			zap.Int("index", i),
			zap.String("role", roleStr),
			zap.String("content", content),
			zap.Int("toolCalls", len(msg.ToolCalls)),
			zap.Strings("toolNames", toolCallNames),
		)
	}

	// 调用LLM
	llmReq := &llm.CompletionRequest{
		Messages: messages,
		Tools:    tools,
	}

	log.Debug("[MainAgent] Calling LLM",
		zap.Int("messageCount", len(messages)),
		zap.Int("toolCount", len(tools)),
	)

	resp, err := m.llm.Complete(ctx, llmReq)
	if err != nil {
		log.Error("[MainAgent] LLM completion failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("llm completion failed: %w", err)
	}

	log.Debug("[MainAgent] LLM response received",
		zap.String("content", truncateForLog(resp.Content, 300)),
		zap.Int("toolCalls", len(resp.ToolCalls)),
		zap.String("finishReason", string(resp.FinishReason)),
		zap.Int("promptTokens", resp.Usage.PromptTokens),
		zap.Int("completionTokens", resp.Usage.CompletionTokens),
		zap.Int("totalTokens", resp.Usage.TotalTokens),
	)

	// 解析响应
	agentResp, err := m.parseResponse(resp)
	if err != nil {
		log.Error("[MainAgent] parseResponse failed",
			zap.Error(err),
		)
		return nil, err
	}

	log.Debug("[MainAgent] Execute completed",
		zap.String("nextAction", agentResp.NextAction.String()),
		zap.String("content", truncateForLog(agentResp.Content, 200)),
		zap.Int("toolCalls", len(agentResp.ToolCalls)),
		zap.Int("subAgentCalls", len(agentResp.SubAgentCalls)),
	)

	return agentResp, nil
}

// truncateForLog 截断日志字符串
func truncateForLog(s string, maxLen int) string {
	if len(s) == 0 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// buildMessages 构建消息列表
func (m *MainAgent) buildMessages(systemPrompt string, req *AgentRequest) []llm.Message {
	messages := []llm.Message{
		llm.NewSystemMessage(systemPrompt),
	}

	// 添加对话历史
	if req.Context != nil && len(req.Context.History) > 0 {
		messages = append(messages, req.Context.History...)
	}

	// 添加用户输入（如果不为空且不是已存在于 history 中的）
	// 通过 Metadata 中的 pending_user_input 标记判断
	if req.UserInput != "" {
		pendingInput, _ := req.Context.Metadata["pending_user_input"].(string)
		if pendingInput != "" && pendingInput != req.UserInput {
			// 标记的输入与实际不同，才添加（这种情况不应该发生）
			messages = append(messages, llm.NewUserMessage(req.UserInput))
		}
		// 如果 pending_input 与 UserInput 相同，说明是从 history 提取的，不添加
	}

	return messages
}

// parseResponse 解析LLM响应
func (m *MainAgent) parseResponse(resp *llm.CompletionResponse) (*AgentResponse, error) {
	log := m.getLogger()

	log.Debug("[MainAgent] parseResponse started",
		zap.String("content", truncateForLog(resp.Content, 200)),
		zap.Int("toolCalls", len(resp.ToolCalls)),
	)

	agentResp := &AgentResponse{
		Content: resp.Content,
	}

	// 处理Tool调用
	if len(resp.ToolCalls) > 0 {
		log.Debug("[MainAgent] Tool calls detected",
			zap.Int("count", len(resp.ToolCalls)),
		)

		// 打印每个tool call
		for i, tc := range resp.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			log.Debug("[MainAgent] Tool call",
				zap.Int("index", i),
				zap.String("toolName", tc.Name),
				zap.String("toolCallID", tc.ID),
				zap.String("arguments", truncateForLog(string(argsJSON), 200)),
			)
		}

		// 检查是否有子Agent调用
		subAgentCalls := m.extractSubAgentCalls(resp.ToolCalls)
		if len(subAgentCalls) > 0 {
			log.Debug("[MainAgent] SubAgent calls extracted",
				zap.Int("count", len(subAgentCalls)),
			)
			for i, sac := range subAgentCalls {
				log.Debug("[MainAgent] SubAgent call",
					zap.Int("index", i),
					zap.String("agentName", sac.AgentName),
					zap.String("intent", truncateForLog(sac.Intent, 100)),
				)
			}
			agentResp.SubAgentCalls = subAgentCalls
			agentResp.NextAction = ActionCallSubAgent
			return agentResp, nil
		}

		// 普通Tool调用
		log.Debug("[MainAgent] Regular tool calls",
			zap.Int("count", len(resp.ToolCalls)),
		)
		agentResp.ToolCalls = resp.ToolCalls
		agentResp.NextAction = ActionContinue
		return agentResp, nil
	}

	// 判断下一步动作
	if resp.Content != "" {
		log.Debug("[MainAgent] Response content ready for player",
			zap.String("content", truncateForLog(resp.Content, 200)),
		)
		agentResp.NextAction = ActionRespondToPlayer
	} else {
		log.Debug("[MainAgent] No content, waiting for input")
		agentResp.NextAction = ActionWaitForInput
	}

	return agentResp, nil
}

// extractSubAgentCalls 从ToolCalls中提取子Agent调用
// 当tool name匹配子Agent名称时，将其视为子Agent调用而非普通Tool调用
func (m *MainAgent) extractSubAgentCalls(toolCalls []llm.ToolCall) []SubAgentCall {
	var subAgentCalls []SubAgentCall

	for _, call := range toolCalls {
		// 检查是否匹配已注册的子Agent
		if _, ok := m.subAgents[call.Name]; ok {
			// 提取意图：优先使用LLM传入的intent参数，否则使用description或tool name
			intent := ""
			if v, ok := call.Arguments["intent"]; ok {
				if s, ok := v.(string); ok {
					intent = s
				}
			}
			if intent == "" {
				if v, ok := call.Arguments["description"]; ok {
					if s, ok := v.(string); ok {
						intent = s
					}
				}
			}
			if intent == "" {
				intent = call.Name
			}

			subAgentCalls = append(subAgentCalls, SubAgentCall{
				AgentName: call.Name,
				Intent:    intent,
			})
		} else {
			// 非子Agent调用，作为普通Tool调用返回
			// 注意：如果混合了子Agent和普通Tool，这里简化处理，
			// 优先处理子Agent，普通Tool在下一轮处理
			return nil
		}
	}

	return subAgentCalls
}

// prepareTemplateData 准备提示词模板数据
func (m *MainAgent) prepareTemplateData(ctx *AgentContext) map[string]any {
	data := make(map[string]any)

	// 游戏会话ID
	if ctx.GameID != "" {
		data["GameID"] = ctx.GameID
	} else {
		data["GameID"] = "未设置"
	}

	// 玩家ID
	if ctx.PlayerID != "" {
		data["PlayerID"] = ctx.PlayerID
	} else {
		data["PlayerID"] = "未设置"
	}

	// 游戏状态
	if ctx.CurrentState != nil {
		data["GameState"] = game_summary.FormatForLLM(ctx.CurrentState)
	} else {
		data["GameState"] = "游戏尚未开始"
	}

	// 可用Tools
	tools := m.registry.GetAllTools()
	toolInfo := make([]map[string]string, 0, len(tools))
	for _, t := range tools {
		toolInfo = append(toolInfo, map[string]string{
			"Name":        t.Name(),
			"Description": t.Description(),
		})
	}
	data["AvailableTools"] = toolInfo

	// 可用子Agent
	subAgentInfo := make([]map[string]string, 0, len(m.subAgents))
	for _, agent := range m.subAgents {
		subAgentInfo = append(subAgentInfo, map[string]string{
			"Name":        agent.Name(),
			"Description": agent.Description(),
		})
	}
	data["SubAgents"] = subAgentInfo

	return data
}

// defaultSystemPrompt 默认系统提示词
func (m *MainAgent) defaultSystemPrompt(ctx *AgentContext) string {
	var parts []string

	parts = append(parts, "你是一位经验丰富的地下城主(Dungeon Master)。")
	parts = append(parts, "核心原则：所有规则判定必须通过调用Tools完成，不得自行计算。")

	// 游戏会话信息
	parts = append(parts, "")
	parts = append(parts, fmt.Sprintf("游戏会话ID: %s", ctx.GameID))
	parts = append(parts, fmt.Sprintf("玩家ID: %s", ctx.PlayerID))
	parts = append(parts, "重要：在调用任何Tool时，必须使用上述ID来标识当前游戏和玩家。")

	if ctx.CurrentState != nil {
		parts = append(parts, "")
		parts = append(parts, "当前游戏状态:")
		parts = append(parts, game_summary.FormatForLLM(ctx.CurrentState))
	}

	parts = append(parts, "")
	parts = append(parts, "可用Tools:")
	for _, t := range m.registry.GetAllTools() {
		parts = append(parts, fmt.Sprintf("- `%s`: %s", t.Name(), t.Description()))
	}

	return strings.Join(parts, "\n")
}
