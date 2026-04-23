package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ============================================================================
// CombatDM Agent 专用工具
// 这些工具封装了 dnd-core 的组合型战斗API，供 CombatDMAgent 的 LLM 通过
// function calling 自主调用。
// ============================================================================

// ExecuteTurnActionTool 执行当前回合的战斗动作
type ExecuteTurnActionTool struct {
	EngineTool
}

func NewExecuteTurnActionTool(e *engine.Engine) *ExecuteTurnActionTool {
	return &ExecuteTurnActionTool{
		EngineTool: *NewEngineTool(
			"execute_turn_action",
			`执行当前回合角色的一个战斗动作。传入 action_id（从 next_turn_with_actions 或上一次 execute_turn_action 返回的动作列表中获取）和可选的 target_id。
返回：动作叙事、剩余可用动作(remaining_actions)、回合是否完成(turn_complete)、战斗是否结束(combat_end)。
当 turn_complete=true 时，应调用 next_turn_with_actions 推进回合。`,
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "执行动作的角色ID（当前回合的行动者）",
					},
					"action_id": map[string]any{
						"type":        "string",
						"description": "要执行的动作ID（从可用动作列表中获取）",
					},
					"target_id": map[string]any{
						"type":        "string",
						"description": "目标角色ID（攻击/法术等需要目标的动作必填）",
					},
					"target_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "多目标动作的目标ID列表（如AOE法术）",
					},
				},
				"required": []string{"game_id", "actor_id", "action_id"},
			},
			e,
			false,
		),
	}
}

func (t *ExecuteTurnActionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actionID, err := RequireString(params, "action_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.ExecuteTurnActionRequest{
		GameID:   model.ID(gameIDStr),
		ActorID:  model.ID(actorIDStr),
		ActionID: actionID,
	}

	// 可选: 单目标
	if targetIDStr := OptionalString(params, "target_id", ""); targetIDStr != "" {
		req.TargetID = model.ID(targetIDStr)
	}

	// 可选: 多目标
	if targetStrs := OptionalStringArray(params, "target_ids"); len(targetStrs) > 0 {
		req.TargetIDs = make([]model.ID, len(targetStrs))
		for i, s := range targetStrs {
			req.TargetIDs[i] = model.ID(s)
		}
	}

	result, err := e.ExecuteTurnAction(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	msg := result.Narrative
	if msg == "" {
		msg = fmt.Sprintf("执行了 %s", result.ActionName)
	}
	if result.TurnComplete {
		msg += "\n[turn_complete=true，请调用 next_turn_with_actions 推进回合]"
	}
	if result.CombatEnd != nil {
		msg += fmt.Sprintf("\n[combat_end: reason=%s, winners=%s]", result.CombatEnd.Reason, result.CombatEnd.Winners)
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: msg,
	}, nil
}

// NextTurnWithActionsTool 推进到下一回合并获取增强回合信息
type NextTurnWithActionsTool struct {
	EngineTool
}

func NewNextTurnWithActionsTool(e *engine.Engine) *NextTurnWithActionsTool {
	return &NextTurnWithActionsTool{
		EngineTool: *NewEngineTool(
			"next_turn_with_actions",
			`推进到下一个角色的回合，返回增强版回合信息：行动者信息（ID、名称、类型、HP、AC、状态）、可用动作列表、所有参战者状态快照。
当 execute_turn_action 返回 turn_complete=true 后调用此工具。
如果返回 combat_end 字段，说明战斗已结束。`,
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
				},
				"required": []string{"game_id"},
			},
			e,
			false,
		),
	}
}

func (t *NextTurnWithActionsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	result, err := e.NextTurnWithActions(ctx, engine.NextTurnRequest{
		GameID: model.ID(gameIDStr),
	})
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	turn := result.Turn
	if turn == nil {
		return &ToolResult{Success: true, Data: result, Message: "回合推进完成"}, nil
	}

	if turn.CombatEnd != nil {
		return &ToolResult{
			Success: true,
			Data:    result,
			Message: fmt.Sprintf("[combat_end: reason=%s, winners=%s]", turn.CombatEnd.Reason, turn.CombatEnd.Winners),
		}, nil
	}

	msg := fmt.Sprintf("第%d轮 — %s(%s)的回合 | HP %d/%d AC %d",
		turn.Round, turn.ActorName, turn.ActorType,
		turn.ActorHP, turn.ActorMaxHP, turn.ActorAC,
	)

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: msg,
	}, nil
}
