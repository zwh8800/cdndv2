package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/prompt"
	"github.com/zwh8800/cdndv2/game_engine/state"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// CombatAgent 战斗管理Agent
type CombatAgent struct {
	registry     *tool.ToolRegistry
	llm          llm.LLMClient
	systemPrompt string
}

// NewCombatAgent 创建战斗管理Agent
func NewCombatAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *CombatAgent {
	return &CombatAgent{
		registry: registry,
		llm:      llmClient,
	}
}

// Name 返回Agent名称
func (a *CombatAgent) Name() string {
	return SubAgentNameCombat
}

// Description 返回Agent描述
func (a *CombatAgent) Description() string {
	return "战斗管理Agent，负责战斗初始化、回合管理、攻击、伤害治疗"
}

// SystemPrompt 返回系统提示词
func (a *CombatAgent) SystemPrompt(ctx *AgentContext) string {
	if a.systemPrompt != "" {
		return a.systemPrompt
	}

	templateStr, err := prompt.LoadSystemPrompt("combat_system.md")
	if err != nil {
		return a.defaultSystemPrompt(ctx)
	}

	data := a.prepareTemplateData(ctx)
	rendered, err := prompt.RenderTemplate(templateStr, data)
	if err != nil {
		return a.defaultSystemPrompt(ctx)
	}

	a.systemPrompt = rendered
	return rendered
}

// Tools 返回Agent可用的Tools
func (a *CombatAgent) Tools() []tool.Tool {
	return a.registry.GetByAgent(SubAgentNameCombat)
}

// Execute 执行Agent逻辑
func (a *CombatAgent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
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
func (a *CombatAgent) CanHandle(intent string) bool {
	keywords := []string{
		"attack", "combat", "turn", "damage", "heal", "move",
		"start_combat", "end_combat", "next_turn", "execute_attack",
		"战斗", "攻击", "回合", "伤害", "治疗", "移动",
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
func (a *CombatAgent) Priority() int {
	return 20
}

// Dependencies 返回依赖的其他Agent
func (a *CombatAgent) Dependencies() []string {
	return []string{SubAgentNameCharacter}
}

// buildMessages 构建消息列表
func (a *CombatAgent) buildMessages(systemPrompt string, req *AgentRequest) []llm.Message {
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
func (a *CombatAgent) parseResponse(resp *llm.CompletionResponse) (*AgentResponse, error) {
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
func (a *CombatAgent) prepareTemplateData(ctx *AgentContext) map[string]any {
	data := make(map[string]any)

	if ctx.CurrentState != nil {
		data["GameState"] = state.FormatForLLM(ctx.CurrentState)
	} else {
		data["GameState"] = "游戏尚未开始"
	}

	tools := a.registry.GetByAgent(SubAgentNameCombat)
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
func (a *CombatAgent) defaultSystemPrompt(ctx *AgentContext) string {
	var parts []string

	parts = append(parts, "你是D&D 5e战斗系统专家。")
	parts = append(parts, "核心原则：所有战斗操作必须通过调用Tools完成，不得自行计算。")

	if ctx.CurrentState != nil {
		parts = append(parts, "")
		parts = append(parts, "当前游戏状态:")
		parts = append(parts, state.FormatForLLM(ctx.CurrentState))
	}

	parts = append(parts, "")
	parts = append(parts, "可用Tools:")
	for _, t := range a.registry.GetByAgent(SubAgentNameCombat) {
		parts = append(parts, fmt.Sprintf("- `%s`: %s", t.Name(), t.Description()))
	}

	return strings.Join(parts, "\n")
}
