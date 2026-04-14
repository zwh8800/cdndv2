package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/zwh8800/cdndv2/game_engine/game_summary"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/prompt"
)

// RouterAgent 路由决策Agent
// 不实现Agent接口，作为ReActLoop的内部组件
type RouterAgent struct {
	llm    llm.LLMClient
	agents map[string]SubAgent
	logger *zap.Logger
}

// NewRouterAgent 创建路由Agent
func NewRouterAgent(llmClient llm.LLMClient, agents map[string]SubAgent) *RouterAgent {
	return &RouterAgent{
		llm:    llmClient,
		agents: agents,
		logger: zap.NewNop(),
	}
}

// SetLogger 设置日志器
func (r *RouterAgent) SetLogger(log *zap.Logger) {
	if log != nil {
		r.logger = log
	}
}

// getLogger 获取日志器
func (r *RouterAgent) getLogger() *zap.Logger {
	if r.logger == nil {
		r.logger = zap.NewNop()
	}
	return r.logger
}

// Route 路由决策
func (r *RouterAgent) Route(ctx context.Context, userInput string, history []llm.Message, gameState *game_summary.GameSummary) (*RouterDecision, error) {
	log := r.getLogger()
	log.Debug("[RouterAgent] Route started",
		zap.String("userInput", userInput),
		zap.Int("historyLength", len(history)),
	)

	// 构建系统提示词
	systemPrompt := r.buildSystemPrompt(gameState)

	// 组装消息
	messages := []llm.Message{llm.NewSystemMessage(systemPrompt)}
	messages = append(messages, history...)
	messages = append(messages, llm.NewUserMessage(userInput))

	// 构建工具定义（可选的输出格式）
	tools := r.buildRouterTools()

	// 调用LLM
	req := &llm.CompletionRequest{
		Messages: messages,
		Tools:    tools,
	}

	resp, err := r.llm.Complete(ctx, req)
	if err != nil {
		log.Error("[RouterAgent] LLM completion failed", zap.Error(err))
		return nil, fmt.Errorf("router llm failed: %w", err)
	}

	// 解析路由决策
	decision, err := r.parseRouterResponse(resp)
	if err != nil {
		log.Error("[RouterAgent] Failed to parse response", zap.Error(err))
		return nil, err
	}

	log.Debug("[RouterAgent] Route completed",
		zap.Int("targetAgentCount", len(decision.TargetAgents)),
		zap.String("executionMode", string(decision.ExecutionMode)),
		zap.String("reasoning", decision.Reasoning),
	)

	return decision, nil
}

// buildSystemPrompt 构建路由系统提示词
func (r *RouterAgent) buildSystemPrompt(gameState *game_summary.GameSummary) string {
	templateStr, err := prompt.LoadSystemPrompt("router_system.md")
	if err != nil {
		return r.defaultSystemPrompt(gameState)
	}

	data := r.prepareTemplateData(gameState)
	rendered, err := prompt.RenderTemplate(templateStr, data)
	if err != nil {
		return r.defaultSystemPrompt(gameState)
	}

	return rendered
}

// buildRouterTools 构建路由工具定义
func (r *RouterAgent) buildRouterTools() []map[string]any {
	return []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        "route_decision",
				"description": "输出路由决策，指定要调用的Agent和执行模式",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"target_agents": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"agent_name": map[string]any{
										"type":        "string",
										"enum":        []string{"character_agent", "combat_agent", "rules_agent"},
										"description": "要委托的Agent名称",
									},
									"intent": map[string]any{
										"type":        "string",
										"description": "传递给Agent的任务意图",
									},
								},
								"required": []string{"agent_name", "intent"},
							},
						},
						"execution_mode": map[string]any{
							"type":        "string",
							"enum":        []string{"sequential", "parallel"},
							"description": "执行模式：parallel(并行)或sequential(串行)",
						},
						"reasoning": map[string]any{
							"type":        "string",
							"description": "路由决策的理由",
						},
						"direct_response": map[string]any{
							"type":        "string",
							"description": "如果不需要调用Agent，直接回复玩家（可选）",
						},
					},
					"required": []string{"target_agents", "execution_mode", "reasoning"},
				},
			},
		},
	}
}

// parseRouterResponse 解析路由响应
func (r *RouterAgent) parseRouterResponse(resp *llm.CompletionResponse) (*RouterDecision, error) {
	decision := &RouterDecision{
		ExecutionMode: ExecutionSequential,
	}

	// 检查是否有工具调用
	if len(resp.ToolCalls) > 0 {
		for _, tc := range resp.ToolCalls {
			if tc.Name == "route_decision" {
				return r.parseRouteDecisionTool(tc.Arguments)
			}
		}
	}

	// 如果没有工具调用，尝试从内容解析JSON
	if resp.Content != "" {
		return r.parseRouteDecisionJSON(resp.Content)
	}

	// 默认：无目标Agent，需要直接响应
	decision.DirectResponse = "请告诉我您想做什么？"
	return decision, nil
}

// parseRouteDecisionTool 从工具调用参数解析路由决策
func (r *RouterAgent) parseRouteDecisionTool(args map[string]any) (*RouterDecision, error) {
	decision := &RouterDecision{
		ExecutionMode: ExecutionSequential,
	}

	if v, ok := args["target_agents"].([]any); ok {
		for _, agent := range v {
			if agentMap, ok := agent.(map[string]any); ok {
				delegation := AgentDelegation{}
				if name, ok := agentMap["agent_name"].(string); ok {
					delegation.AgentName = name
				}
				if intent, ok := agentMap["intent"].(string); ok {
					delegation.Intent = intent
				}
				decision.TargetAgents = append(decision.TargetAgents, delegation)
			}
		}
	}

	if mode, ok := args["execution_mode"].(string); ok {
		decision.ExecutionMode = ExecutionMode(mode)
	}
	if reasoning, ok := args["reasoning"].(string); ok {
		decision.Reasoning = reasoning
	}
	if directResp, ok := args["direct_response"].(string); ok {
		decision.DirectResponse = directResp
	}

	return decision, nil
}

// parseRouteDecisionJSON 从JSON内容解析路由决策
func (r *RouterAgent) parseRouteDecisionJSON(content string) (*RouterDecision, error) {
	// 尝试提取JSON
	var jsonContent string
	start := findJSONStart(content)
	if start >= 0 {
		end := findJSONEnd(content, start)
		if end > start {
			jsonContent = content[start : end+1]
		}
	}

	if jsonContent == "" {
		// 无法解析，返回默认决策
		return &RouterDecision{
			ExecutionMode:  ExecutionSequential,
			DirectResponse: content,
		}, nil
	}

	var decision RouterDecision
	if err := json.Unmarshal([]byte(jsonContent), &decision); err != nil {
		r.getLogger().Debug("[RouterAgent] Failed to parse JSON", zap.Error(err))
		return &RouterDecision{
			ExecutionMode:  ExecutionSequential,
			DirectResponse: content,
		}, nil
	}

	return &decision, nil
}

// findJSONStart 查找JSON起始位置
func findJSONStart(s string) int {
	for i, c := range s {
		if c == '{' {
			return i
		}
	}
	return -1
}

// findJSONEnd 查找JSON结束位置
func findJSONEnd(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// prepareTemplateData 准备模板数据
func (r *RouterAgent) prepareTemplateData(gameState *game_summary.GameSummary) map[string]any {
	data := make(map[string]any)

	// 游戏状态
	if gameState != nil {
		data["GameState"] = game_summary.FormatForLLM(gameState)
	} else {
		data["GameState"] = "游戏尚未开始"
	}

	// 可用Agent
	agentInfo := make([]map[string]any, 0, len(r.agents))
	for _, agent := range r.agents {
		deps := agent.Dependencies()
		depStr := "无"
		if len(deps) > 0 {
			depStr = fmt.Sprintf("%v", deps)
		}
		agentInfo = append(agentInfo, map[string]any{
			"Name":         agent.Name(),
			"Description":  agent.Description(),
			"Priority":     agent.Priority(),
			"Dependencies": depStr,
		})
	}
	data["Agents"] = agentInfo

	return data
}

// defaultSystemPrompt 默认路由提示词
func (r *RouterAgent) defaultSystemPrompt(gameState *game_summary.GameSummary) string {
	var parts []string

	parts = append(parts, "你是任务路由专家。你的任务是分析玩家输入，决定应该调用哪些专业Agent。")
	parts = append(parts, "")
	parts = append(parts, "## 可用Agent")

	for _, agent := range r.agents {
		parts = append(parts, fmt.Sprintf("- **%s**: %s (优先级: %d)", agent.Name(), agent.Description(), agent.Priority()))
	}

	if gameState != nil {
		parts = append(parts, "")
		parts = append(parts, "## 当前游戏状态")
		parts = append(parts, game_summary.FormatForLLM(gameState))
	}

	parts = append(parts, "")
	parts = append(parts, "## 输出格式")
	parts = append(parts, "调用 route_decision 工具输出你的决策。")

	return fmt.Sprintf("%s", parts)
}
