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
	registry *tool.ToolRegistry
	llm      llm.LLMClient
	logger   *zap.Logger
}

// NewMainAgent 创建主Agent
func NewMainAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *MainAgent {
	return &MainAgent{
		registry: registry,
		llm:      llmClient,
		logger:   zap.NewNop(),
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

// Tools 返回Agent可用的Tools（只读工具 + delegate_task）
func (m *MainAgent) Tools() []tool.Tool {
	readOnlyTools := m.registry.GetReadOnlyTools()
	if dt, ok := m.registry.Get(tool.DelegateTaskToolName); ok {
		readOnlyTools = append(readOnlyTools, dt)
	}
	return readOnlyTools
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

	// 获取Tools定义 - 只暴露只读工具和 delegate_task
	tools := m.registry.GetReadOnlySchemas()
	if dt, ok := m.registry.Get(tool.DelegateTaskToolName); ok {
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        dt.Name(),
				"description": dt.Description(),
				"parameters":  dt.ParametersSchema(),
			},
		})
	}
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
		Usage:   resp.Usage,
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

		// 分类工具调用：delegate_task、只读工具、写操作工具
		delegateCalls, readOnlyCalls, writeCalls := m.separateCallsByAccess(resp.ToolCalls)

		// 写操作调用转为 delegate_task（防御性拦截）
		if len(writeCalls) > 0 {
			log.Warn("LLM attempted to call write tools directly, converting to delegate_task",
				zap.Int("writeCallCount", len(writeCalls)),
			)
			for _, wc := range writeCalls {
				agentName := m.inferAgentForTool(wc.Name)
				delegateCalls = append(delegateCalls, llm.ToolCall{
					ID:   wc.ID,
					Name: tool.DelegateTaskToolName,
					Arguments: map[string]any{
						"agent_name": agentName,
						"intent":     fmt.Sprintf("执行 %s 操作", wc.Name),
					},
				})
			}
		}

		// 处理 delegate_task 调用
		if len(delegateCalls) > 0 {
			log.Debug("[MainAgent] Delegate task calls detected",
				zap.Int("count", len(delegateCalls)),
			)
			agentResp.DelegateToolCalls = delegateCalls
			agentResp.SubAgentCalls = m.convertToSubAgentCalls(delegateCalls)
			// 只读工具调用也保留，和委托并行执行
			agentResp.ToolCalls = readOnlyCalls
			agentResp.NextAction = ActionDelegate
			return agentResp, nil
		}

		// 只有只读工具调用
		log.Debug("[MainAgent] Read-only tool calls",
			zap.Int("count", len(readOnlyCalls)),
		)
		agentResp.ToolCalls = readOnlyCalls
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

// separateCallsByAccess 将工具调用分为三类：delegate_task、只读工具、写操作工具
func (m *MainAgent) separateCallsByAccess(toolCalls []llm.ToolCall) (delegateCalls, readOnlyCalls, writeCalls []llm.ToolCall) {
	for _, call := range toolCalls {
		if tool.IsDelegateTaskTool(call.Name) {
			delegateCalls = append(delegateCalls, call)
			continue
		}
		t, ok := m.registry.Get(call.Name)
		if !ok {
			// 未知工具，保留原样让 ToolRegistry 返回错误
			readOnlyCalls = append(readOnlyCalls, call)
			continue
		}
		if t.ReadOnly() {
			readOnlyCalls = append(readOnlyCalls, call)
		} else {
			writeCalls = append(writeCalls, call)
		}
	}
	return
}

// convertToSubAgentCalls 将 delegate_task 工具调用转换为 SubAgentCall
func (m *MainAgent) convertToSubAgentCalls(delegateCalls []llm.ToolCall) []SubAgentCall {
	var calls []SubAgentCall
	for _, call := range delegateCalls {
		agentName, intent, _ := tool.ExtractDelegation(call.Arguments)
		if agentName != "" && intent != "" {
			calls = append(calls, SubAgentCall{
				AgentName: agentName,
				Intent:    intent,
			})
		}
	}
	return calls
}

// inferAgentForTool 根据工具名推断应该委托给哪个 Agent
func (m *MainAgent) inferAgentForTool(toolName string) string {
	// 查询 registry 中的 byAgent 映射
	agents := m.registry.GetAgentsForTool(toolName)
	// 优先返回非 MainAgent 的 Agent
	for _, a := range agents {
		if a != MainAgentName {
			return a
		}
	}
	// 默认委托给 rules_agent
	return SubAgentNameRules
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

	// 只读工具信息（MainAgent 可直接调用）
	readOnlyTools := m.registry.GetReadOnlyTools()
	toolInfo := make([]map[string]string, 0, len(readOnlyTools))
	for _, t := range readOnlyTools {
		toolInfo = append(toolInfo, map[string]string{
			"Name":        t.Name(),
			"Description": t.Description(),
		})
	}
	data["ReadOnlyTools"] = toolInfo
	data["AvailableTools"] = toolInfo // 兼容旧模板

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
	parts = append(parts, "可用Tools（只读查询）:")
	for _, t := range m.registry.GetReadOnlyTools() {
		parts = append(parts, fmt.Sprintf("- `%s`: %s", t.Name(), t.Description()))
	}
	parts = append(parts, "")
	parts = append(parts, "对于修改游戏状态的操作（创建角色、战斗、施法等），请使用 delegate_task 委托给专业 Agent。")

	return strings.Join(parts, "\n")
}
