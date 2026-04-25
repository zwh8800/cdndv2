package tool

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zwh8800/cdndv2/game_engine/llm"
)

type fakePlanActionTool struct {
	name    string
	calls   *[]string
	success bool
}

func (t fakePlanActionTool) Name() string {
	return t.name
}

func (t fakePlanActionTool) Description() string {
	return "fake action"
}

func (t fakePlanActionTool) ParametersSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{"value": map[string]any{"type": "string"}},
		"required":   []string{"value"},
	}
}

func (t fakePlanActionTool) ReadOnly() bool {
	return false
}

func (t fakePlanActionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	value, _ := params["value"].(string)
	*t.calls = append(*t.calls, fmt.Sprintf("%s:%s", t.name, value))
	if !t.success {
		return &ToolResult{Success: false, Error: "planned failure"}, nil
	}
	return &ToolResult{Success: true, Message: "ok " + value}, nil
}

func TestSubmitCombatPlanExecutesActionsSequentially(t *testing.T) {
	registry := NewToolRegistry()
	var calls []string
	registry.Register(fakePlanActionTool{name: "first_action", calls: &calls, success: true}, []string{"combat_agent"}, "test")
	registry.Register(fakePlanActionTool{name: "second_action", calls: &calls, success: true}, []string{"combat_agent"}, "test")

	planTool := NewSubmitCombatPlanTool(registry, "combat_agent")
	registry.Register(planTool, []string{"combat_agent"}, "test")

	result, err := planTool.Execute(context.Background(), map[string]any{
		"plan_type": "full_round",
		"actions": []map[string]any{
			{"tool": "first_action", "params": map[string]any{"value": "a"}, "reason": "first"},
			{"tool": "second_action", "params": map[string]any{"value": "b"}, "reason": "second"},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if got, want := strings.Join(calls, ","), "first_action:a,second_action:b"; got != want {
		t.Fatalf("actions executed out of order: got %q want %q", got, want)
	}
	if !strings.Contains(result.Message, "2 个动作全部成功") {
		t.Fatalf("expected aggregate success message, got %q", result.Message)
	}
}

func TestSubmitCombatPlanRejectsUnknownTool(t *testing.T) {
	registry := NewToolRegistry()
	planTool := NewSubmitCombatPlanTool(registry, "combat_agent")

	result, err := planTool.Execute(context.Background(), map[string]any{
		"plan_type": "full_round",
		"actions": []map[string]any{
			{"tool": "missing_action", "params": map[string]any{"value": "a"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Success || !strings.Contains(result.Error, "tool not found: missing_action") {
		t.Fatalf("expected unknown tool failure, got success=%v error=%q", result.Success, result.Error)
	}
}

func TestSubmitCombatPlanRejectsUnauthorizedTool(t *testing.T) {
	registry := NewToolRegistry()
	var calls []string
	registry.Register(fakePlanActionTool{name: "other_action", calls: &calls, success: true}, []string{"other_agent"}, "test")
	planTool := NewSubmitCombatPlanTool(registry, "combat_agent")

	result, err := planTool.Execute(context.Background(), map[string]any{
		"plan_type": "full_round",
		"actions": []map[string]any{
			{"tool": "other_action", "params": map[string]any{"value": "a"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Success || !strings.Contains(result.Error, "tool not allowed for agent") {
		t.Fatalf("expected unauthorized tool failure, got success=%v error=%q", result.Success, result.Error)
	}
	if len(calls) != 0 {
		t.Fatalf("unauthorized action should not execute, got calls %v", calls)
	}
}

func TestSubmitCombatPlanStopsAfterFailedAction(t *testing.T) {
	registry := NewToolRegistry()
	var calls []string
	registry.Register(fakePlanActionTool{name: "first_action", calls: &calls, success: true}, []string{"combat_agent"}, "test")
	registry.Register(fakePlanActionTool{name: "failing_action", calls: &calls, success: false}, []string{"combat_agent"}, "test")
	registry.Register(fakePlanActionTool{name: "never_action", calls: &calls, success: true}, []string{"combat_agent"}, "test")
	planTool := NewSubmitCombatPlanTool(registry, "combat_agent")

	result, err := planTool.Execute(context.Background(), map[string]any{
		"plan_type": "full_round",
		"actions": []map[string]any{
			{"tool": "first_action", "params": map[string]any{"value": "a"}},
			{"tool": "failing_action", "params": map[string]any{"value": "b"}},
			{"tool": "never_action", "params": map[string]any{"value": "c"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected failure")
	}
	if got, want := strings.Join(calls, ","), "first_action:a,failing_action:b"; got != want {
		t.Fatalf("expected execution to stop after failure: got %q want %q", got, want)
	}
	if !strings.Contains(result.Error, "战斗计划执行中止") || !strings.Contains(result.Error, "planned failure") {
		t.Fatalf("expected aggregate failure details, got %q", result.Error)
	}
}

func TestSubmitCombatPlanFailureKeepsExecutedResultsInRegistryContent(t *testing.T) {
	registry := NewToolRegistry()
	var calls []string
	registry.Register(fakePlanActionTool{name: "first_action", calls: &calls, success: true}, []string{"combat_agent"}, "test")
	registry.Register(fakePlanActionTool{name: "failing_action", calls: &calls, success: false}, []string{"combat_agent"}, "test")
	registry.Register(NewSubmitCombatPlanTool(registry, "combat_agent"), []string{"combat_agent"}, "test")

	results := registry.ExecuteToolsForAgent(context.Background(), "combat_agent", []llm.ToolCall{
		{
			ID:   "plan_call",
			Name: "submit_combat_plan",
			Arguments: map[string]any{
				"plan_type": "full_round",
				"actions": []map[string]any{
					{"tool": "first_action", "params": map[string]any{"value": "a"}},
					{"tool": "failing_action", "params": map[string]any{"value": "b"}},
				},
			},
		},
	})

	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if !results[0].IsError {
		t.Fatalf("expected registry result to be an error")
	}
	if !strings.Contains(results[0].Content, "executed_actions") || !strings.Contains(results[0].Content, "first_action") {
		t.Fatalf("expected rendered error to include executed action data, got %q", results[0].Content)
	}
}
