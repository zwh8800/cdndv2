package tool

import (
	"context"
)

// DelegateTaskTool 专门用于调用SubAgent的工具
// LLM通过标准function calling调用此工具，ReAct Loop拦截并路由到SubAgent
// 此工具不会被ToolRegistry真正执行，而是在PhaseAct中被拦截处理
type DelegateTaskTool struct {
	BaseTool
}

// DelegateTaskToolName 工具名称常量
const DelegateTaskToolName = "delegate_task"

// NewDelegateTaskTool 创建委托任务工具
func NewDelegateTaskTool() *DelegateTaskTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_name": map[string]any{
				"type":        "string",
				"enum":        []string{"character_agent", "combat_agent", "rules_agent"},
				"description": "要委托的Agent名称：character_agent(角色管理)、combat_agent(战斗管理)、rules_agent(规则检定)",
			},
			"intent": map[string]any{
				"type":        "string",
				"description": "传递给Agent的任务意图描述，清晰说明需要Agent完成什么任务",
			},
			"input": map[string]any{
				"type":        "string",
				"description": "额外的具体输入信息，如参数、选项等",
			},
		},
		"required": []string{"agent_name", "intent"},
	}

	return &DelegateTaskTool{
		BaseTool: BaseTool{
			name:        DelegateTaskToolName,
			description: "将任务委托给专门的Agent处理。用于角色管理、战斗操作、规则检定等专业任务。当你需要执行复杂的游戏操作时，使用此工具委托给对应的专家Agent。",
			schema:      schema,
		},
	}
}

// Execute 执行委托任务
// 注意：此方法不会被真正调用，ReActLoop会在PhaseAct中拦截delegate_task调用
func (t *DelegateTaskTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	// 此Tool不会真正被ToolRegistry执行
	// ReAct Loop在PhaseAct中拦截delegate_task调用，路由到SubAgent
	return &ToolResult{
		Success: false,
		Error:   "delegate_task should be intercepted by ReActLoop, not executed by ToolRegistry",
	}, nil
}

// IsDelegateTaskTool 检查是否为委托任务工具调用
func IsDelegateTaskTool(toolName string) bool {
	return toolName == DelegateTaskToolName
}

// ExtractDelegation 从ToolCall参数中提取AgentDelegation
func ExtractDelegation(toolCall map[string]any) (agentName, intent, input string) {
	if v, ok := toolCall["agent_name"].(string); ok {
		agentName = v
	}
	if v, ok := toolCall["intent"].(string); ok {
		intent = v
	}
	if v, ok := toolCall["input"].(string); ok {
		input = v
	}
	return
}
