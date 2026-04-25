package gameengine

import (
	"context"
	"strings"
	"testing"

	"github.com/zwh8800/cdndv2/game_engine/agent"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

type fakeSubAgent struct {
	name      string
	registry  *tool.ToolRegistry
	responses []*agent.AgentResponse
	index     int
}

func (a *fakeSubAgent) Name() string {
	return a.name
}

func (a *fakeSubAgent) Description() string {
	return "fake sub agent"
}

func (a *fakeSubAgent) SystemPrompt(ctx *agent.AgentContext) string {
	return ""
}

func (a *fakeSubAgent) Tools() []tool.Tool {
	if a.registry == nil {
		return nil
	}
	return a.registry.GetByAgent(a.name)
}

func (a *fakeSubAgent) Execute(ctx context.Context, req *agent.AgentRequest) (*agent.AgentResponse, error) {
	if a.index >= len(a.responses) {
		return &agent.AgentResponse{Content: "done"}, nil
	}
	resp := a.responses[a.index]
	a.index++
	return resp, nil
}

func (a *fakeSubAgent) CanHandle(intent string) bool {
	return true
}

func (a *fakeSubAgent) Priority() int {
	return 1
}

func (a *fakeSubAgent) Dependencies() []string {
	return nil
}

func (a *fakeSubAgent) ToolsForTask(task string) []tool.Tool {
	return a.Tools()
}

type fakeLoopActionTool struct {
	calls *[]string
}

func (t fakeLoopActionTool) Name() string {
	return "record_action"
}

func (t fakeLoopActionTool) Description() string {
	return "record action"
}

func (t fakeLoopActionTool) ParametersSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{"value": map[string]any{"type": "string"}},
		"required":   []string{"value"},
	}
}

func (t fakeLoopActionTool) Execute(ctx context.Context, params map[string]any) (*tool.ToolResult, error) {
	value, _ := params["value"].(string)
	*t.calls = append(*t.calls, value)
	return &tool.ToolResult{Success: true, Message: "recorded " + value}, nil
}

func (t fakeLoopActionTool) ReadOnly() bool {
	return false
}

func TestExecuteSingleDelegationExecutesSubmittedCombatPlanAndContinues(t *testing.T) {
	registry := tool.NewToolRegistry()
	var calls []string
	registry.Register(fakeLoopActionTool{calls: &calls}, []string{agent.SubAgentNameCombat}, "test")
	registry.Register(tool.NewSubmitCombatPlanTool(registry, agent.SubAgentNameCombat), []string{agent.SubAgentNameCombat}, "test")

	combat := &fakeSubAgent{
		name:     agent.SubAgentNameCombat,
		registry: registry,
		responses: []*agent.AgentResponse{
			{
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_plan",
						Name: "submit_combat_plan",
						Arguments: map[string]any{
							"plan_type": "full_round",
							"actions": []any{
								map[string]any{"tool": "record_action", "params": map[string]any{"value": "first"}},
								map[string]any{"tool": "record_action", "params": map[string]any{"value": "second"}},
							},
						},
					},
				},
			},
			{Content: "战斗计划已经结算。"},
		},
	}

	loop := NewReActLoop(nil, nil, nil, map[string]agent.SubAgent{agent.SubAgentNameCombat: combat}, registry, nil, 5)
	loop.state.agentContext = agent.NewAgentContext("game", "player", nil)

	result := loop.executeSingleDelegation(context.Background(), agent.SubAgentCall{
		AgentName: agent.SubAgentNameCombat,
		Intent:    "执行完整战斗回合",
	})

	if !result.Success {
		t.Fatalf("expected delegation success, got error %q", result.Error)
	}
	if result.Content != "战斗计划已经结算。" {
		t.Fatalf("expected final response after tool result, got %q", result.Content)
	}
	if got, want := strings.Join(calls, ","), "first,second"; got != want {
		t.Fatalf("expected submitted plan actions to execute: got %q want %q", got, want)
	}
}

func TestExecuteSingleDelegationDoesNotParseCombatPlanFromText(t *testing.T) {
	registry := tool.NewToolRegistry()
	var calls []string
	registry.Register(fakeLoopActionTool{calls: &calls}, []string{agent.SubAgentNameCombat}, "test")
	registry.Register(tool.NewSubmitCombatPlanTool(registry, agent.SubAgentNameCombat), []string{agent.SubAgentNameCombat}, "test")

	content := "```json\n{\"plan_type\":\"full_round\",\"actions\":[{\"tool\":\"record_action\",\"params\":{\"value\":\"should_not_run\"}}]}\n```"
	combat := &fakeSubAgent{
		name:     agent.SubAgentNameCombat,
		registry: registry,
		responses: []*agent.AgentResponse{
			{Content: content},
		},
	}

	loop := NewReActLoop(nil, nil, nil, map[string]agent.SubAgent{agent.SubAgentNameCombat: combat}, registry, nil, 5)
	loop.state.agentContext = agent.NewAgentContext("game", "player", nil)

	result := loop.executeSingleDelegation(context.Background(), agent.SubAgentCall{
		AgentName: agent.SubAgentNameCombat,
		Intent:    "执行完整战斗回合",
	})

	if !result.Success {
		t.Fatalf("expected delegation success, got error %q", result.Error)
	}
	if result.Content != content {
		t.Fatalf("expected plain text response to pass through unchanged, got %q", result.Content)
	}
	if len(calls) != 0 {
		t.Fatalf("plain text JSON must not execute actions, got calls %v", calls)
	}
}

func TestExecuteSingleDelegationStillExecutesRegularCombatToolCall(t *testing.T) {
	registry := tool.NewToolRegistry()
	var calls []string
	registry.Register(fakeLoopActionTool{calls: &calls}, []string{agent.SubAgentNameCombat}, "test")

	combat := &fakeSubAgent{
		name:     agent.SubAgentNameCombat,
		registry: registry,
		responses: []*agent.AgentResponse{
			{
				ToolCalls: []llm.ToolCall{
					{
						ID:        "call_action",
						Name:      "record_action",
						Arguments: map[string]any{"value": "single"},
					},
				},
			},
			{Content: "单个战斗动作已经结算。"},
		},
	}

	loop := NewReActLoop(nil, nil, nil, map[string]agent.SubAgent{agent.SubAgentNameCombat: combat}, registry, nil, 5)
	loop.state.agentContext = agent.NewAgentContext("game", "player", nil)

	result := loop.executeSingleDelegation(context.Background(), agent.SubAgentCall{
		AgentName: agent.SubAgentNameCombat,
		Intent:    "执行一次攻击",
	})

	if !result.Success {
		t.Fatalf("expected delegation success, got error %q", result.Error)
	}
	if result.Content != "单个战斗动作已经结算。" {
		t.Fatalf("expected final response after regular tool result, got %q", result.Content)
	}
	if got, want := strings.Join(calls, ","), "single"; got != want {
		t.Fatalf("expected regular tool call to execute: got %q want %q", got, want)
	}
}

func TestExecuteSingleDelegationFailsWhenSubAgentNeverFinalizes(t *testing.T) {
	registry := tool.NewToolRegistry()
	var calls []string
	registry.Register(fakeLoopActionTool{calls: &calls}, []string{agent.SubAgentNameCombat}, "test")

	responses := make([]*agent.AgentResponse, 0, subAgentMaxIterations)
	for i := 0; i < subAgentMaxIterations; i++ {
		responses = append(responses, &agent.AgentResponse{
			ToolCalls: []llm.ToolCall{
				{
					ID:        "call_action",
					Name:      "record_action",
					Arguments: map[string]any{"value": "loop"},
				},
			},
		})
	}

	combat := &fakeSubAgent{
		name:      agent.SubAgentNameCombat,
		registry:  registry,
		responses: responses,
	}

	loop := NewReActLoop(nil, nil, nil, map[string]agent.SubAgent{agent.SubAgentNameCombat: combat}, registry, nil, 5)
	loop.state.agentContext = agent.NewAgentContext("game", "player", nil)

	result := loop.executeSingleDelegation(context.Background(), agent.SubAgentCall{
		AgentName: agent.SubAgentNameCombat,
		Intent:    "一直调用工具但不给最终文本",
	})

	if result.Success {
		t.Fatalf("expected max-iteration delegation failure, got content %q", result.Content)
	}
	if !strings.Contains(result.Error, "最大迭代次数") {
		t.Fatalf("expected max-iteration error, got %q", result.Error)
	}
}

func TestResolveCombatPlanActionNamesNormalizesTypedActions(t *testing.T) {
	loop := NewReActLoop(nil, nil, nil, nil, tool.NewToolRegistry(), nil, 5)
	args := map[string]any{
		"plan_type": "full_round",
		"actions": []tool.CombatPlanAction{
			{Tool: "record_action", Params: map[string]any{"value": "typed"}},
		},
	}

	loop.resolveToolNames(context.Background(), []llm.ToolCall{
		{
			ID:        "call_plan",
			Name:      "submit_combat_plan",
			Arguments: args,
		},
	})

	actions, ok := args["actions"].([]any)
	if !ok {
		t.Fatalf("expected typed plan actions to be normalized to []any, got %T", args["actions"])
	}
	action, ok := actions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected normalized action map, got %T", actions[0])
	}
	if action["tool"] != "record_action" {
		t.Fatalf("expected normalized tool name, got %v", action["tool"])
	}
}
