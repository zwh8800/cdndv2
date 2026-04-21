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

// SubAgentConfig 子Agent配置，定义各Agent独有的属性
type SubAgentConfig struct {
	Name         string   // Agent名称常量
	Description  string   // Agent描述
	TemplateFile string   // 系统提示词模板文件名（如 "combat_system.md"）
	DomainIntro  string   // 默认提示词中的领域介绍（如 "你是D&D 5e战斗系统专家。"）
	DomainRule   string   // 核心原则（如 "所有战斗操作必须通过调用Tools完成，不得自行计算。"）
	KeyRules     []string // 关键规则列表
	Priority     int      // 处理优先级
	Dependencies []string // 依赖的其他Agent名称
	Keywords     []string // CanHandle关键词列表
	// ExtraTemplateData 用于动态注入额外的模板数据，返回的键值对会合并到模板数据中
	ExtraTemplateData func(ctx *AgentContext) map[string]any
}

// BaseSubAgent 子Agent基类，提供公共逻辑
type BaseSubAgent struct {
	config   SubAgentConfig
	registry *tool.ToolRegistry
	llm      llm.LLMClient
	logger   *zap.Logger
}

// NewBaseSubAgent 创建子Agent基类
func NewBaseSubAgent(config SubAgentConfig, registry *tool.ToolRegistry, llmClient llm.LLMClient) *BaseSubAgent {
	return &BaseSubAgent{
		config:   config,
		registry: registry,
		llm:      llmClient,
		logger:   zap.NewNop(),
	}
}

// SetLogger 设置日志器
func (a *BaseSubAgent) SetLogger(log *zap.Logger) {
	if log != nil {
		a.logger = log
	}
}

// getLogger 获取日志器
func (a *BaseSubAgent) getLogger() *zap.Logger {
	if a.logger == nil {
		a.logger = zap.NewNop()
	}
	return a.logger
}

// Config 返回Agent配置
func (a *BaseSubAgent) Config() SubAgentConfig {
	return a.config
}

// Registry 返回Tool注册表
func (a *BaseSubAgent) Registry() *tool.ToolRegistry {
	return a.registry
}

// Name 返回Agent名称
func (a *BaseSubAgent) Name() string {
	return a.config.Name
}

// Description 返回Agent描述
func (a *BaseSubAgent) Description() string {
	return a.config.Description
}

// SystemPrompt 返回系统提示词
func (a *BaseSubAgent) SystemPrompt(ctx *AgentContext) string {
	templateStr, err := prompt.LoadSystemPrompt(a.config.TemplateFile)
	if err != nil {
		return a.defaultSystemPrompt(ctx)
	}

	data := a.prepareTemplateData(ctx)
	rendered, err := prompt.RenderTemplate(templateStr, data)
	if err != nil {
		return a.defaultSystemPrompt(ctx)
	}

	return rendered
}

// Tools 返回Agent可用的Tools
func (a *BaseSubAgent) Tools() []tool.Tool {
	return a.registry.GetByAgent(a.config.Name)
}

// Execute 执行Agent逻辑
func (a *BaseSubAgent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	log := a.getLogger()
	agentName := a.config.Name

	log.Debug(fmt.Sprintf("[%s] Execute started", agentName),
		zap.String("userInput", req.UserInput),
		zap.Int("historyLength", len(req.Context.History)),
	)

	systemPrompt := a.SystemPrompt(req.Context)
	log.Debug(fmt.Sprintf("[%s] System prompt built", agentName),
		zap.Int("promptLength", len(systemPrompt)),
	)

	messages := a.buildMessages(systemPrompt, req)
	log.Debug(fmt.Sprintf("[%s] Messages built", agentName),
		zap.Int("messageCount", len(messages)),
	)

	tools := a.toolsToLLMFormat(req.Intent)
	log.Debug(fmt.Sprintf("[%s] Tools retrieved", agentName),
		zap.Int("toolCount", len(tools)),
	)

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
		log.Debug(fmt.Sprintf("[%s] LLM request message", agentName),
			zap.Int("index", i),
			zap.String("role", roleStr),
			zap.String("content", content),
			zap.Int("toolCalls", len(msg.ToolCalls)),
			zap.Strings("toolNames", toolCallNames),
		)
	}

	llmReq := &llm.CompletionRequest{
		Messages: messages,
		Tools:    tools,
	}

	log.Debug(fmt.Sprintf("[%s] Calling LLM", agentName),
		zap.Int("messageCount", len(messages)),
		zap.Int("toolCount", len(tools)),
	)

	resp, err := a.llm.Complete(ctx, llmReq)
	if err != nil {
		log.Error(fmt.Sprintf("[%s] LLM completion failed", agentName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("llm completion failed: %w", err)
	}

	log.Debug(fmt.Sprintf("[%s] LLM response received", agentName),
		zap.String("content", truncateForLog(resp.Content, 300)),
		zap.Int("toolCalls", len(resp.ToolCalls)),
		zap.String("finishReason", string(resp.FinishReason)),
		zap.Int("promptTokens", resp.Usage.PromptTokens),
		zap.Int("completionTokens", resp.Usage.CompletionTokens),
		zap.Int("totalTokens", resp.Usage.TotalTokens),
	)

	agentResp, err := a.parseResponse(resp)
	if err != nil {
		log.Error(fmt.Sprintf("[%s] parseResponse failed", agentName),
			zap.Error(err),
		)
		return nil, err
	}

	log.Debug(fmt.Sprintf("[%s] Execute completed", agentName),
		zap.String("nextAction", agentResp.NextAction.String()),
		zap.String("content", truncateForLog(agentResp.Content, 200)),
		zap.Int("toolCalls", len(agentResp.ToolCalls)),
	)

	return agentResp, nil
}

// CanHandle 判断是否能处理该意图
func (a *BaseSubAgent) CanHandle(intent string) bool {
	intentLower := strings.ToLower(intent)
	for _, kw := range a.config.Keywords {
		if strings.Contains(intentLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// Priority 返回处理优先级
func (a *BaseSubAgent) Priority() int {
	return a.config.Priority
}

// Dependencies 返回依赖的其他Agent
func (a *BaseSubAgent) Dependencies() []string {
	return a.config.Dependencies
}

// ToolsForTask 默认实现：返回全部工具（具体Agent可覆盖）
func (a *BaseSubAgent) ToolsForTask(task string) []tool.Tool {
	return a.Tools()
}

// buildMessages 构建消息列表
func (a *BaseSubAgent) buildMessages(systemPrompt string, req *AgentRequest) []llm.Message {
	messages := []llm.Message{
		llm.NewSystemMessage(systemPrompt),
	}

	if req.Context != nil && len(req.Context.History) > 0 {
		messages = append(messages, req.Context.History...)
	}

	if req.UserInput != "" {
		messages = append(messages, llm.NewUserMessage(req.UserInput))
	}

	return messages
}

// parseResponse 解析LLM响应
func (a *BaseSubAgent) parseResponse(resp *llm.CompletionResponse) (*AgentResponse, error) {
	log := a.getLogger()
	agentName := a.config.Name

	log.Debug(fmt.Sprintf("[%s] parseResponse started", agentName),
		zap.String("content", truncateForLog(resp.Content, 200)),
		zap.Int("toolCalls", len(resp.ToolCalls)),
	)

	agentResp := &AgentResponse{
		Content: resp.Content,
	}

	if len(resp.ToolCalls) > 0 {
		log.Debug(fmt.Sprintf("[%s] Tool calls detected", agentName),
			zap.Int("count", len(resp.ToolCalls)),
		)

		for i, tc := range resp.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			log.Debug(fmt.Sprintf("[%s] Tool call", agentName),
				zap.Int("index", i),
				zap.String("toolName", tc.Name),
				zap.String("toolCallID", tc.ID),
				zap.String("arguments", truncateForLog(string(argsJSON), 200)),
			)
		}

		agentResp.ToolCalls = resp.ToolCalls
		agentResp.NextAction = ActionContinue
		return agentResp, nil
	}

	if resp.Content != "" {
		log.Debug(fmt.Sprintf("[%s] Response content ready for player", agentName),
			zap.String("content", truncateForLog(resp.Content, 200)),
		)
		agentResp.NextAction = ActionRespondToPlayer
	} else {
		log.Debug(fmt.Sprintf("[%s] No content, waiting for input", agentName))
		agentResp.NextAction = ActionWaitForInput
	}

	return agentResp, nil
}

// prepareTemplateData 准备提示词模板数据
func (a *BaseSubAgent) prepareTemplateData(ctx *AgentContext) map[string]any {
	data := make(map[string]any)

	if ctx.GameID != "" {
		data["GameID"] = ctx.GameID
	} else {
		data["GameID"] = "未设置"
	}

	if ctx.PlayerID != "" {
		data["PlayerID"] = ctx.PlayerID
	} else {
		data["PlayerID"] = "未设置"
	}

	if ctx.CurrentState != nil {
		data["GameState"] = game_summary.FormatForLLM(ctx.CurrentState)
	} else {
		data["GameState"] = "游戏尚未开始"
	}

	tools := a.registry.GetByAgent(a.config.Name)
	toolInfo := make([]map[string]string, 0, len(tools))
	for _, t := range tools {
		toolInfo = append(toolInfo, map[string]string{
			"Name":        t.Name(),
			"Description": t.Description(),
		})
	}
	data["AvailableTools"] = toolInfo

	// 注入已知实体ID，供 SubAgent 间共享
	if len(ctx.KnownEntityIDs) > 0 {
		data["KnownEntityIDs"] = formatKnownEntityIDs(ctx.KnownEntityIDs)
	} else {
		data["KnownEntityIDs"] = ""
	}

	// 调用扩展函数注入额外的模板数据
	if a.config.ExtraTemplateData != nil {
		for k, v := range a.config.ExtraTemplateData(ctx) {
			data[k] = v
		}
	}

	return data
}

// defaultSystemPrompt 默认系统提示词
func (a *BaseSubAgent) defaultSystemPrompt(ctx *AgentContext) string {
	var parts []string

	parts = append(parts, a.config.DomainIntro)
	parts = append(parts, fmt.Sprintf("核心原则：%s", a.config.DomainRule))

	parts = append(parts, "")
	parts = append(parts, fmt.Sprintf("游戏会话ID: %s", ctx.GameID))
	parts = append(parts, fmt.Sprintf("玩家ID: %s", ctx.PlayerID))
	parts = append(parts, "重要：在调用任何Tool时，必须使用上述ID来标识当前游戏和玩家。")

	if len(ctx.KnownEntityIDs) > 0 {
		parts = append(parts, "")
		parts = append(parts, formatKnownEntityIDs(ctx.KnownEntityIDs))
	}

	if ctx.CurrentState != nil {
		parts = append(parts, "")
		parts = append(parts, "当前游戏状态:")
		parts = append(parts, game_summary.FormatForLLM(ctx.CurrentState))
	}

	parts = append(parts, "")
	parts = append(parts, "可用Tools:")
	for _, t := range a.registry.GetByAgent(a.config.Name) {
		parts = append(parts, fmt.Sprintf("- `%s`: %s", t.Name(), t.Description()))
	}

	if len(a.config.KeyRules) > 0 {
		parts = append(parts, "")
		parts = append(parts, "关键规则：")
		for _, rule := range a.config.KeyRules {
			parts = append(parts, fmt.Sprintf("- %s", rule))
		}
	}

	return strings.Join(parts, "\n")
}

// toolsToLLMFormat 将Tools转换为LLM function calling格式
// 使用 ToolsForTask 根据任务描述动态过滤工具
func (a *BaseSubAgent) toolsToLLMFormat(task string) []map[string]any {
	tools := a.ToolsForTask(task)
	if len(tools) == 0 {
		tools = a.Tools()
	}
	result := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  t.ParametersSchema(),
			},
		})
	}
	return result
}

// formatKnownEntityIDs 格式化已知实体ID为可读文本，注入到 SubAgent system prompt
func formatKnownEntityIDs(entityIDs map[string]string) string {
	if len(entityIDs) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "## 已知实体ID（重要：调用 API 时必须使用以下 ID）")
	parts = append(parts, "")

	if actorID, ok := entityIDs["actor_id"]; ok {
		parts = append(parts, fmt.Sprintf("- **角色ID (actor_id)**: `%s`", actorID))
	}
	if sceneID, ok := entityIDs["scene_id"]; ok {
		parts = append(parts, fmt.Sprintf("- **场景ID (scene_id)**: `%s`", sceneID))
	}

	parts = append(parts, "")
	parts = append(parts, "**注意**: 在调用任何需要 actor_id 或 scene_id 的 API 时，必须使用上述 ID 值，不得使用角色名称或其他标识符。")

	return strings.Join(parts, "\n")
}
