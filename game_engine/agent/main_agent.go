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

// MainAgent 主Agent(DM)实现
type MainAgent struct {
	registry         *tool.ToolRegistry
	llm              llm.LLMClient
	subAgents        map[string]SubAgent
	systemPrompt     string
	systemPromptData map[string]any
}

// NewMainAgent 创建主Agent
func NewMainAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient, subAgents map[string]SubAgent) *MainAgent {
	return &MainAgent{
		registry:         registry,
		llm:              llmClient,
		subAgents:        subAgents,
		systemPromptData: make(map[string]any),
	}
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
	if m.systemPrompt != "" {
		return m.systemPrompt
	}

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

	m.systemPrompt = rendered
	return rendered
}

// Tools 返回Agent可用的Tools
func (m *MainAgent) Tools() []tool.Tool {
	return m.registry.GetAllTools()
}

// Execute 执行Agent逻辑
func (m *MainAgent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	// 构建系统提示词
	systemPrompt := m.SystemPrompt(req.Context)

	// 组装消息
	messages := m.buildMessages(systemPrompt, req)

	// 获取Tools定义
	tools := m.registry.GetAll()

	// 调用LLM
	llmReq := &llm.CompletionRequest{
		Messages: messages,
		Tools:    tools,
	}

	resp, err := m.llm.Complete(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("llm completion failed: %w", err)
	}

	// 解析响应
	return m.parseResponse(resp)
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

	// 添加用户输入
	if req.UserInput != "" {
		messages = append(messages, llm.NewUserMessage(req.UserInput))
	}

	return messages
}

// parseResponse 解析LLM响应
func (m *MainAgent) parseResponse(resp *llm.CompletionResponse) (*AgentResponse, error) {
	agentResp := &AgentResponse{
		Content: resp.Content,
	}

	// 处理Tool调用
	if len(resp.ToolCalls) > 0 {
		agentResp.ToolCalls = resp.ToolCalls
		agentResp.NextAction = ActionContinue
		return agentResp, nil
	}

	// 判断下一步动作
	if resp.Content != "" {
		agentResp.NextAction = ActionRespondToPlayer
	} else {
		agentResp.NextAction = ActionWaitForInput
	}

	return agentResp, nil
}

// prepareTemplateData 准备提示词模板数据
func (m *MainAgent) prepareTemplateData(ctx *AgentContext) map[string]any {
	data := make(map[string]any)

	// 游戏状态
	if ctx.CurrentState != nil {
		data["GameState"] = state.FormatForLLM(ctx.CurrentState)
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

	if ctx.CurrentState != nil {
		parts = append(parts, "")
		parts = append(parts, "当前游戏状态:")
		parts = append(parts, state.FormatForLLM(ctx.CurrentState))
	}

	parts = append(parts, "")
	parts = append(parts, "可用Tools:")
	for _, t := range m.registry.GetAllTools() {
		parts = append(parts, fmt.Sprintf("- `%s`: %s", t.Name(), t.Description()))
	}

	return strings.Join(parts, "\n")
}
