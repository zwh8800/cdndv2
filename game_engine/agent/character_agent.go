package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/zwh8800/cdndv2/game_engine/game_summary"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/prompt"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// CharacterAgent 角色管理Agent
type CharacterAgent struct {
	registry *tool.ToolRegistry
	llm      llm.LLMClient
}

// NewCharacterAgent 创建角色管理Agent
func NewCharacterAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *CharacterAgent {
	return &CharacterAgent{
		registry: registry,
		llm:      llmClient,
	}
}

// Name 返回Agent名称
func (a *CharacterAgent) Name() string {
	return SubAgentNameCharacter
}

// Description 返回Agent描述
func (a *CharacterAgent) Description() string {
	return "角色管理Agent，负责角色创建、查询、更新、经验、休息"
}

// SystemPrompt 返回系统提示词
func (a *CharacterAgent) SystemPrompt(ctx *AgentContext) string {
	templateStr, err := prompt.LoadSystemPrompt("character_system.md")
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
func (a *CharacterAgent) Tools() []tool.Tool {
	return a.registry.GetByAgent(SubAgentNameCharacter)
}

// Execute 执行Agent逻辑
func (a *CharacterAgent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	systemPrompt := a.SystemPrompt(req.Context)

	messages := a.buildMessages(systemPrompt, req)

	tools := make([]map[string]any, 0)
	for _, t := range a.Tools() {
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  t.ParametersSchema(),
			},
		})
	}

	llmReq := &llm.CompletionRequest{
		Messages: messages,
		Tools:    tools,
	}

	resp, err := a.llm.Complete(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("llm completion failed: %w", err)
	}

	return a.parseResponse(resp)
}

// CanHandle 判断是否能处理该意图
func (a *CharacterAgent) CanHandle(intent string) bool {
	keywords := []string{
		"create_character", "create_pc", "create_npc", "create_enemy", "create_companion",
		"get_actor", "get_pc", "list_actors", "update_actor", "remove_actor",
		"add_experience", "level_up", "short_rest", "long_rest",
		"character", "角色", "创建", "升级", "经验", "休息",
	}

	intentLower := strings.ToLower(intent)
	for _, kw := range keywords {
		if strings.Contains(intentLower, kw) {
			return true
		}
	}
	return false
}

// Priority 返回处理优先级
func (a *CharacterAgent) Priority() int {
	return 10
}

// Dependencies 返回依赖的其他Agent
func (a *CharacterAgent) Dependencies() []string {
	return nil
}

// buildMessages 构建消息列表
func (a *CharacterAgent) buildMessages(systemPrompt string, req *AgentRequest) []llm.Message {
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
func (a *CharacterAgent) parseResponse(resp *llm.CompletionResponse) (*AgentResponse, error) {
	agentResp := &AgentResponse{
		Content: resp.Content,
	}

	if len(resp.ToolCalls) > 0 {
		agentResp.ToolCalls = resp.ToolCalls
		agentResp.NextAction = ActionContinue
		return agentResp, nil
	}

	if resp.Content != "" {
		agentResp.NextAction = ActionRespondToPlayer
	} else {
		agentResp.NextAction = ActionWaitForInput
	}

	return agentResp, nil
}

// prepareTemplateData 准备提示词模板数据
func (a *CharacterAgent) prepareTemplateData(ctx *AgentContext) map[string]any {
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

	if ctx.CurrentState != nil {
		data["GameState"] = game_summary.FormatForLLM(ctx.CurrentState)
	} else {
		data["GameState"] = "游戏尚未开始"
	}

	tools := a.registry.GetByAgent(SubAgentNameCharacter)
	toolInfo := make([]map[string]string, 0, len(tools))
	for _, t := range tools {
		toolInfo = append(toolInfo, map[string]string{
			"Name":        t.Name(),
			"Description": t.Description(),
		})
	}
	data["AvailableTools"] = toolInfo

	return data
}

// defaultSystemPrompt 默认系统提示词
func (a *CharacterAgent) defaultSystemPrompt(ctx *AgentContext) string {
	var parts []string

	parts = append(parts, "你是D&D 5e角色管理专家。")
	parts = append(parts, "核心原则：所有角色操作必须通过调用Tools完成，不得自行计算。")

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
	for _, t := range a.registry.GetByAgent(SubAgentNameCharacter) {
		parts = append(parts, fmt.Sprintf("- `%s`: %s", t.Name(), t.Description()))
	}

	return strings.Join(parts, "\n")
}
