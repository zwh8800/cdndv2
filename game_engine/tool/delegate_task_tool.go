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
				"type": "string",
				"enum": []string{
					"character_agent",
					"combat_agent",
					"rules_agent",
					"inventory_agent",
					"narrative_agent",
					"npc_agent",
					"memory_agent",
					"movement_agent",
					"mount_agent",
					"crafting_agent",
					"data_query_agent",
				},
				"description": "要委托的Agent名称，根据任务类型选择对应的专业Agent",
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
			description: "将任务委托给专门的Agent处理。根据任务类型选择对应Agent：character_agent(角色创建/更新/经验)、combat_agent(战斗/攻击/伤害/治疗)、rules_agent(检定/豁免/法术/休息)、inventory_agent(物品/装备/货币)、narrative_agent(场景/探索/旅行/陷阱)、npc_agent(NPC交互)、memory_agent(任务/生活方式/时间)、movement_agent(跳跃/坠落/窒息)、mount_agent(骑乘/坐骑)、crafting_agent(制作)、data_query_agent(查询种族/职业/法术等静态数据)。",
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
