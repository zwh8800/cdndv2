package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zwh8800/cdndv2/game_engine/llm"
)

// CombatPlan 战斗计划（Plan-Then-Act 模式）。
type CombatPlan struct {
	PlanType    string             `json:"plan_type"`
	Actions     []CombatPlanAction `json:"actions"`
	Contingency string             `json:"contingency,omitempty"`
}

// CombatPlanAction 战斗计划中的单个动作。
type CombatPlanAction struct {
	Tool   string         `json:"tool"`
	Params map[string]any `json:"params"`
	Reason string         `json:"reason,omitempty"`
}

// SubmitCombatPlanTool 接收显式战斗计划并按顺序调度已注册工具。
type SubmitCombatPlanTool struct {
	BaseTool
	registry  *ToolRegistry
	agentName string
}

func NewSubmitCombatPlanTool(registry *ToolRegistry, agentName string) *SubmitCombatPlanTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"plan_type": map[string]any{
				"type":        "string",
				"enum":        []string{"full_round", "partial", "single_action"},
				"description": "计划类型：完整回合、局部计划或单个动作计划",
			},
			"actions": map[string]any{
				"type":        "array",
				"description": "按顺序执行的战斗工具调用计划",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tool": map[string]any{
							"type":        "string",
							"description": "要调用的已授权战斗工具名",
						},
						"params": map[string]any{
							"type":                 "object",
							"description":          "透传给目标工具的参数对象",
							"additionalProperties": true,
						},
						"reason": map[string]any{
							"type":        "string",
							"description": "选择该动作的战术原因",
						},
					},
					"required": []string{"tool", "params"},
				},
				"minItems": 1,
			},
			"contingency": map[string]any{
				"type":        "string",
				"description": "给 DM 叙事参考的应急说明；系统不会解释或自动执行条件分支",
			},
		},
		"required": []string{"plan_type", "actions"},
	}

	desc := `Submit an explicit combat plan for sequential execution.

Use when: You need to process a full combat round or multiple coordinated combat actions.

Do NOT write the plan as normal text or markdown JSON. Call this tool with the plan as arguments.

Each action must name an existing tool that this combat agent is allowed to call. The plan runner only validates and dispatches tools; all D&D mechanics remain inside dnd-core backed tools.`

	return &SubmitCombatPlanTool{
		BaseTool: BaseTool{
			name:        "submit_combat_plan",
			description: desc,
			schema:      schema,
			readOnly:    false,
		},
		registry:  registry,
		agentName: agentName,
	}
}

func (t *SubmitCombatPlanTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	plan, err := parseCombatPlan(params)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	executionResults := make([]string, 0, len(plan.Actions))
	executed := make([]map[string]any, 0, len(plan.Actions))

	for i, action := range plan.Actions {
		if err := t.validateAction(action); err != nil {
			message := buildCombatPlanSummary(false, plan, executionResults)
			return &ToolResult{
				Success: false,
				Error:   fmt.Sprintf("战斗计划第 %d 个动作无效: %v\n%s", i+1, err, message),
				Data: map[string]any{
					"plan_type":        plan.PlanType,
					"executed_actions": executed,
					"failed_action":    i + 1,
				},
			}, nil
		}

		call := llm.ToolCall{
			ID:        fmt.Sprintf("combat_plan_%d_%s", i+1, action.Tool),
			Name:      action.Tool,
			Arguments: action.Params,
		}
		results := t.registry.executeTools(ctx, t.agentName, []llm.ToolCall{call})
		for _, result := range results {
			line := fmt.Sprintf("[%d/%d] %s: %s", i+1, len(plan.Actions), action.Tool, result.Content)
			executionResults = append(executionResults, line)
			executed = append(executed, map[string]any{
				"index":   i + 1,
				"tool":    action.Tool,
				"reason":  action.Reason,
				"success": !result.IsError,
				"result":  result.Content,
			})
			if result.IsError {
				message := buildCombatPlanSummary(false, plan, executionResults)
				return &ToolResult{
					Success: false,
					Error:   message,
					Data: map[string]any{
						"plan_type":        plan.PlanType,
						"executed_actions": executed,
						"failed_action":    i + 1,
					},
				}, nil
			}
		}
	}

	message := buildCombatPlanSummary(true, plan, executionResults)
	return &ToolResult{
		Success: true,
		Message: message,
		Data: map[string]any{
			"plan_type":        plan.PlanType,
			"executed_actions": executed,
		},
	}, nil
}

func parseCombatPlan(params map[string]any) (*CombatPlan, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("无法读取战斗计划参数: %w", err)
	}

	var plan CombatPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil, fmt.Errorf("战斗计划参数格式错误: %w", err)
	}
	if plan.PlanType == "" {
		return nil, fmt.Errorf("缺少 plan_type")
	}
	if len(plan.Actions) == 0 {
		return nil, fmt.Errorf("actions 不能为空")
	}
	return &plan, nil
}

func (t *SubmitCombatPlanTool) validateAction(action CombatPlanAction) error {
	if action.Tool == "" {
		return fmt.Errorf("缺少 tool")
	}
	if action.Tool == t.Name() {
		return fmt.Errorf("submit_combat_plan 不能递归调用自身")
	}
	if len(action.Params) == 0 {
		return fmt.Errorf("params 不能为空")
	}
	if _, ok := t.registry.Get(action.Tool); !ok {
		return fmt.Errorf("tool not found: %s", action.Tool)
	}
	if !t.registry.IsToolAllowedForAgent(t.agentName, action.Tool) {
		return fmt.Errorf("tool not allowed for agent: %s cannot call %s", t.agentName, action.Tool)
	}
	return nil
}

func buildCombatPlanSummary(allSucceeded bool, plan *CombatPlan, executionResults []string) string {
	var b strings.Builder
	if allSucceeded {
		fmt.Fprintf(&b, "战斗计划执行完成 (%d 个动作全部成功)。", len(plan.Actions))
	} else {
		b.WriteString("战斗计划执行中止。")
	}
	if plan.Contingency != "" {
		fmt.Fprintf(&b, "\n应急计划: %s", plan.Contingency)
	}
	if len(executionResults) > 0 {
		b.WriteString("\n执行结果:\n")
		b.WriteString(strings.Join(executionResults, "\n"))
	}
	return b.String()
}
