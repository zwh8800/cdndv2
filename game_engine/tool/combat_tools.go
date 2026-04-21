package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// StartCombatTool 开始战斗
type StartCombatTool struct {
	EngineTool
}

func NewStartCombatTool(e *engine.Engine) *StartCombatTool {
	return &StartCombatTool{
		EngineTool: *NewEngineTool(
			"start_combat",
			"开始一场战斗遭遇",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"scene_id": map[string]any{
						"type":        "string",
						"description": "战斗发生的场景ID",
					},
					"participant_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "参战者ID列表",
					},
				},
				"required": []string{"game_id", "scene_id", "participant_ids"},
			},
			e,
			false,
		),
	}
}

func (t *StartCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	sceneIDStr, err := RequireString(params, "scene_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	participantStrs, err := RequireStringArray(params, "participant_ids")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	sceneID := model.ID(sceneIDStr)
	pids := make([]model.ID, len(participantStrs))
	for i, pid := range participantStrs {
		pids[i] = model.ID(pid)
	}

	req := engine.StartCombatRequest{
		GameID:         gameID,
		SceneID:        sceneID,
		ParticipantIDs: pids,
	}

	result, err := e.StartCombat(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.Combat,
		Message: "战斗开始！先攻顺序已确定",
	}, nil
}

// StartCombatWithSurpriseTool 开始带突袭的战斗
type StartCombatWithSurpriseTool struct {
	EngineTool
}

func NewStartCombatWithSurpriseTool(e *engine.Engine) *StartCombatWithSurpriseTool {
	return &StartCombatWithSurpriseTool{
		EngineTool: *NewEngineTool(
			"start_combat_with_surprise",
			"开始一场带突袭的战斗",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"scene_id": map[string]any{
						"type":        "string",
						"description": "战斗发生的场景ID",
					},
					"stealthy_side": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "潜行方角色ID列表（不会被突袭）",
					},
					"observers": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "被观察方角色ID列表（被突袭）",
					},
				},
				"required": []string{"game_id", "scene_id", "stealthy_side", "observers"},
			},
			e,
			false,
		),
	}
}

func (t *StartCombatWithSurpriseTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	sceneIDStr, err := RequireString(params, "scene_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	stealthyStrs, err := RequireStringArray(params, "stealthy_side")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	observerStrs, err := RequireStringArray(params, "observers")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	sceneID := model.ID(sceneIDStr)
	stealthy := make([]model.ID, len(stealthyStrs))
	for i, id := range stealthyStrs {
		stealthy[i] = model.ID(id)
	}
	observers := make([]model.ID, len(observerStrs))
	for i, id := range observerStrs {
		observers[i] = model.ID(id)
	}

	req := engine.StartCombatWithSurpriseRequest{
		GameID:       gameID,
		SceneID:      sceneID,
		StealthySide: stealthy,
		Observers:    observers,
	}

	result, err := e.StartCombatWithSurprise(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.Combat,
		Message: "突袭战斗开始！潜行方获得突袭优势",
	}, nil
}

// GetCurrentCombatTool 获取当前战斗状态
type GetCurrentCombatTool struct {
	EngineTool
}

func NewGetCurrentCombatTool(e *engine.Engine) *GetCurrentCombatTool {
	return &GetCurrentCombatTool{
		EngineTool: *NewEngineTool(
			"get_current_combat",
			"获取当前进行中的战斗完整状态，包括所有参战者、先攻顺序、回合信息等。Use when: 需要了解战斗全局态势（还有谁活着、先攻顺序如何）；玩家询问当前战斗情况。Do NOT use when: 只需要知道当前轮到谁（用 get_current_turn）；战斗尚未开始（应先调用 start_combat）。",
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
			true,
		),
	}
}

func (t *GetCurrentCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	req := engine.GetCurrentCombatRequest{
		GameID: gameID,
	}

	result, err := e.GetCurrentCombat(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.Combat,
		Message: "当前战斗状态",
	}, nil
}

// GetCurrentTurnTool 获取当前回合信息
type GetCurrentTurnTool struct {
	EngineTool
}

func NewGetCurrentTurnTool(e *engine.Engine) *GetCurrentTurnTool {
	return &GetCurrentTurnTool{
		EngineTool: *NewEngineTool(
			"get_current_turn",
			"获取当前轮到哪个角色的回合信息，包括角色ID、角色名称、回合状态等。Use when: 需要确认当前谁该行动；玩家询问'现在轮到谁了'。Do NOT use when: 需要完整战斗状态（用 get_current_combat）；需要推进到下一回合（用 next_turn，这是 write 操作）。",
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
			true,
		),
	}
}

func (t *GetCurrentTurnTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	req := engine.GetCurrentTurnRequest{
		GameID: gameID,
	}

	result, err := e.GetCurrentTurn(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: "当前是 " + result.ActorName + " 的回合",
	}, nil
}

// NextTurnTool 推进到下一回合
type NextTurnTool struct {
	EngineTool
}

func NewNextTurnTool(e *engine.Engine) *NextTurnTool {
	return &NextTurnTool{
		EngineTool: *NewEngineTool(
			"next_turn",
			"推进到下一个角色的回合",
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

func (t *NextTurnTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	req := engine.NextTurnRequest{
		GameID: gameID,
	}

	result, err := e.NextTurn(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.Combat,
		Message: "回合推进到 " + result.Combat.CurrentTurn.ActorName,
	}, nil
}

// ExecuteActionTool 执行战斗动作
type ExecuteActionTool struct {
	EngineTool
}

func NewExecuteActionTool(e *engine.Engine) *ExecuteActionTool {
	return &ExecuteActionTool{
		EngineTool: *NewEngineTool(
			"execute_action",
			"执行一个战斗动作（如冲刺、脱离、闪避等）",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "角色ID",
					},
					"action_type": map[string]any{
						"type":        "string",
						"description": "动作类型 (attack, dash, disengage, dodge, help, hide, ready, search)",
					},
				},
				"required": []string{"game_id", "actor_id", "action_type"},
			},
			e,
			false,
		),
	}
}

func (t *ExecuteActionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actionTypeStr, err := RequireString(params, "action_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)
	actionType := model.ActionType(actionTypeStr)

	req := engine.ExecuteActionRequest{
		GameID:  gameID,
		ActorID: actorID,
		Action: engine.ActionInput{
			Type: actionType,
		},
	}

	result, err := e.ExecuteAction(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.ActionResult.Message,
	}, nil
}

// ExecuteAttackTool 执行攻击
type ExecuteAttackTool struct {
	EngineTool
}

func NewExecuteAttackTool(e *engine.Engine) *ExecuteAttackTool {
	return &ExecuteAttackTool{
		EngineTool: *NewEngineTool(
			"execute_attack",
			"执行一次攻击",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"attacker_id": map[string]any{
						"type":        "string",
						"description": "攻击者ID",
					},
					"target_id": map[string]any{
						"type":        "string",
						"description": "目标ID",
					},
					"weapon_id": map[string]any{
						"type":        "string",
						"description": "武器ID（可选）",
					},
					"is_unarmed": map[string]any{
						"type":        "boolean",
						"description": "是否徒手攻击",
					},
					"is_off_hand": map[string]any{
						"type":        "boolean",
						"description": "是否副手攻击",
					},
					"advantage": map[string]any{
						"type":        "string",
						"enum":        []string{"none", "advantage", "disadvantage"},
						"description": "优势/劣势",
					},
				},
				"required": []string{"game_id", "attacker_id", "target_id"},
			},
			e,
			false,
		),
	}
}

func (t *ExecuteAttackTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	attackerIDStr, err := RequireString(params, "attacker_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	targetIDStr, err := RequireString(params, "target_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	attackerID := model.ID(attackerIDStr)
	targetID := model.ID(targetIDStr)

	attack := engine.AttackInput{}

	if wid := OptionalString(params, "weapon_id", ""); wid != "" {
		id := model.ID(wid)
		attack.WeaponID = &id
	}
	attack.IsUnarmed = OptionalBool(params, "is_unarmed", false)
	attack.IsOffHand = OptionalBool(params, "is_off_hand", false)
	adv := OptionalString(params, "advantage", "none")
	switch adv {
	case "advantage":
		attack.Advantage = model.RollModifier{Advantage: true}
	case "disadvantage":
		attack.Advantage = model.RollModifier{Disadvantage: true}
	}

	req := engine.ExecuteAttackRequest{
		GameID:     gameID,
		AttackerID: attackerID,
		TargetID:   targetID,
		Attack:     attack,
	}

	result, err := e.ExecuteAttack(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.AttackResult,
		Message: result.AttackResult.Message,
	}, nil
}

// MoveActorTool 移动角色
type MoveActorTool struct {
	EngineTool
}

func NewMoveActorTool(e *engine.Engine) *MoveActorTool {
	return &MoveActorTool{
		EngineTool: *NewEngineTool(
			"move_actor",
			"在战斗中移动角色",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "角色ID",
					},
					"x": map[string]any{
						"type":        "integer",
						"description": "目标X坐标",
					},
					"y": map[string]any{
						"type":        "integer",
						"description": "目标Y坐标",
					},
				},
				"required": []string{"game_id", "actor_id", "x", "y"},
			},
			e,
			false,
		),
	}
}

func (t *MoveActorTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	x, err := RequireInt(params, "x")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	y, err := RequireInt(params, "y")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)

	req := engine.MoveActorRequest{
		GameID:  gameID,
		ActorID: actorID,
		To: model.Point{
			X: x,
			Y: y,
		},
	}

	result, err := e.MoveActor(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.MoveResult.Message,
	}, nil
}

// ExecuteDamageTool 施加伤害
type ExecuteDamageTool struct {
	EngineTool
}

func NewExecuteDamageTool(e *engine.Engine) *ExecuteDamageTool {
	return &ExecuteDamageTool{
		EngineTool: *NewEngineTool(
			"execute_damage",
			"对目标施加伤害",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"target_id": map[string]any{
						"type":        "string",
						"description": "目标ID",
					},
					"amount": map[string]any{
						"type":        "integer",
						"description": "伤害量",
					},
					"damage_type": map[string]any{
						"type":        "string",
						"description": "伤害类型",
					},
				},
				"required": []string{"game_id", "target_id", "amount"},
			},
			e,
			false,
		),
	}
}

func (t *ExecuteDamageTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	targetIDStr, err := RequireString(params, "target_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	amount, err := RequireInt(params, "amount")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	targetID := model.ID(targetIDStr)

	damage := engine.DamageInput{
		Amount: amount,
	}

	dt := OptionalString(params, "damage_type", "")
	if dt != "" {
		damage.Type = model.DamageType(dt)
	}

	req := engine.ExecuteDamageRequest{
		GameID:   gameID,
		TargetID: targetID,
		Damage:   damage,
	}

	result, err := e.ExecuteDamage(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.DamageResult,
		Message: result.DamageResult.Message,
	}, nil
}

// ExecuteHealingTool 治疗
type ExecuteHealingTool struct {
	EngineTool
}

func NewExecuteHealingTool(e *engine.Engine) *ExecuteHealingTool {
	return &ExecuteHealingTool{
		EngineTool: *NewEngineTool(
			"execute_healing",
			"治疗目标",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"target_id": map[string]any{
						"type":        "string",
						"description": "目标ID",
					},
					"amount": map[string]any{
						"type":        "integer",
						"description": "治疗量",
					},
				},
				"required": []string{"game_id", "target_id", "amount"},
			},
			e,
			false,
		),
	}
}

func (t *ExecuteHealingTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	targetIDStr, err := RequireString(params, "target_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	amount, err := RequireInt(params, "amount")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	targetID := model.ID(targetIDStr)

	req := engine.ExecuteHealingRequest{
		GameID:   gameID,
		TargetID: targetID,
		Amount:   amount,
	}

	result, err := e.ExecuteHealing(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.Message,
	}, nil
}

// PerformDeathSaveTool 死亡豁免检定
type PerformDeathSaveTool struct {
	EngineTool
}

func NewPerformDeathSaveTool(e *engine.Engine) *PerformDeathSaveTool {
	return &PerformDeathSaveTool{
		EngineTool: *NewEngineTool(
			"perform_death_save",
			"执行死亡豁免检定",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "角色ID",
					},
				},
				"required": []string{"game_id", "actor_id"},
			},
			e,
			false,
		),
	}
}

func (t *PerformDeathSaveTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)

	req := engine.SavingThrowRequest{
		GameID:  gameID,
		ActorID: actorID,
		Ability: model.AbilityConstitution,
		DC:      10,
	}

	result, err := e.PerformSavingThrow(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	msg := fmt.Sprintf("死亡豁免检定: 掷骰=%d", result.RollTotal)
	if result.Success {
		msg += "，成功！"
	} else {
		msg += "，失败..."
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: msg,
	}, nil
}

// EndCombatTool 结束战斗
type EndCombatTool struct {
	EngineTool
}

func NewEndCombatTool(e *engine.Engine) *EndCombatTool {
	return &EndCombatTool{
		EngineTool: *NewEngineTool(
			"end_combat",
			"结束当前战斗",
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

func (t *EndCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	req := engine.EndCombatRequest{
		GameID: gameID,
	}

	err = e.EndCombat(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: "战斗结束",
	}, nil
}

// ========== 复合工具 - 战斗系统 ==========

// NewCombatAttackTool 复合攻击工具：自动获取可见目标 → 执行攻击 → 自动计算伤害
//
// 合并: list_visible_actors + execute_attack + execute_damage
func NewCombatAttackTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"game_id": map[string]any{
				"type":        "string",
				"description": "游戏会话ID，系统自动注入，无需用户提供",
			},
			"attacker_id": map[string]any{
				"type":        "string",
				"description": "攻击者ID",
			},
			"target_name": map[string]any{
				"type":        "string",
				"description": "目标角色名称（支持名称解析）",
			},
			"weapon_name": map[string]any{
				"type":        "string",
				"description": "使用的武器名称（可选）",
			},
			"attack_type": map[string]any{
				"type":        "string",
				"enum":        []string{"melee", "ranged"},
				"description": "攻击类型：melee(近战) 或 ranged(远程)",
			},
			"is_unarmed": map[string]any{
				"type":        "boolean",
				"description": "是否徒手攻击",
				"default":     false,
			},
			"is_off_hand": map[string]any{
				"type":        "boolean",
				"description": "是否副手攻击",
				"default":     false,
			},
			"advantage": map[string]any{
				"type":        "string",
				"enum":        []string{"none", "advantage", "disadvantage"},
				"description": "优势/劣势，默认 none",
				"default":     "none",
			},
		},
		"required": []string{"game_id", "attacker_id", "target_name", "attack_type"},
	}

	desc := `Execute a complete attack in combat: automatically find visible target → roll attack → apply damage if hit.

Use when: You need to make an attack against an enemy in combat. One call completes the entire attack process.

Do NOT use when: You just need to check combat status (use show_combat_status instead).

Parameters:
  - attacker_id: ID of the attacking actor (attacker)
  - target_name: Name of the target to attack (name will be auto-resolved to ID)
  - attack_type: 'melee' or 'ranged'
  - weapon_name: Optional name of the weapon to use
  - is_unarmed: Whether this is an unarmed attack
  - is_off_hand: Whether this is an off-hand attack
  - advantage: Roll with advantage or disadvantage

Returns: Complete attack result including attack roll, whether hit, damage dealt, and target HP status.`

	steps := []ToolStep{
		{
			ToolName: "get_actors_in_current_combat_participants",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				// Actually we already have attacker_id from user params
				return map[string]any{
					"game_id": params["game_id"],
				}
			},
		},
		{
			ToolName: "execute_attack",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				// In composite schema we get target_id from name resolution
				// For now, we assume the registry will handle name resolution upstream
				out := map[string]any{
					"game_id":      params["game_id"],
					"attacker_id":  params["attacker_id"],
					"target_id":    params["target_id"], // will be resolved by upstream name resolver
					"attack_type":  params["attack_type"],
					"is_unarmed":   params["is_unarmed"],
					"is_off_hand":  params["is_off_hand"],
					"advantage":     params["advantage"],
				}
				if wn, ok := params["weapon_name"]; ok {
					out["weapon_name"] = wn
				}
				return out
			},
		},
	}

	return NewCompositeTool(
		"combat_attack",
		desc,
		schema,
		registry,
		steps,
		false, // not read only (modifies game state)
	)
}

// NewCombatStartTool 复合开始战斗工具：初始化战斗 → 推进到第一回合 → 返回战斗状态
//
// 合并: start_combat + next_turn
func NewCombatStartTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"game_id": map[string]any{
				"type":        "string",
				"description": "游戏会话ID",
			},
			"scene_id": map[string]any{
				"type":        "string",
				"description": "战斗发生的场景ID",
			},
			"participant_ids": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "参战者ID列表",
			},
			"has_surprise": map[string]any{
				"type":        "boolean",
				"description": "是否有突袭（潜行一方突袭观察者）",
				"default":     false,
			},
			"stealthy_side": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "有突袭时：潜行方ID列表",
			},
			"observers": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "有突袭时：被观察方ID列表",
			},
		},
		"required": []string{"game_id", "scene_id", "participant_ids"},
	}

	desc := `Start a new combat encounter and advance to the first turn.

Use when: Players are starting a new combat encounter. One call initializes combat and gets to the first actor's turn.

Do NOT use when: Combat is already ongoing.

Steps automatically: 1) Initialize combat with initiative rolls 2) Advance to first turn 3) Return complete combat status.

Parameters:
  - game_id: Game session ID
  - scene_id: Scene where combat occurs
  - participant_ids: List of all participant IDs
  - has_surprise: Whether one side surprises the other
  - stealthy_side: If surprise, list of stealthy participants
  - observers: If surprise, list of surprised observers

Returns: Full combat status with initiative order and current turn.`

	steps := []ToolStep{
		{
			ToolName: "start_combat",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				if hasSurprise, ok := params["has_surprise"].(bool); ok && hasSurprise {
					return map[string]any{
						"game_id":        params["game_id"],
						"scene_id":      params["scene_id"],
						"stealthy_side": params["stealthy_side"],
						"observers":     params["observers"],
					}
				}
				return map[string]any{
					"game_id":         params["game_id"],
					"scene_id":        params["scene_id"],
					"participant_ids": params["participant_ids"],
				}
			},
		},
		{
			ToolName: "next_turn",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return map[string]any{
					"game_id": params["game_id"],
				}
			},
		},
	}

	return NewCompositeTool(
		"combat_start",
		desc,
		schema,
		registry,
		steps,
		false,
	)
}

// NewCombatHealTool 复合治疗工具：执行治疗 → 更新目标HP → 返回结果
//
// 合并: execute_healing (already single step, but enhanced with richer result
func NewCombatHealTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"game_id": map[string]any{
				"type":        "string",
				"description": "游戏会话ID",
			},
			"target_id": map[string]any{
				"type":        "string",
				"description": "接受治疗的目标ID",
			},
			"amount": map[string]any{
				"type":        "integer",
				"description": "治疗量",
			},
		},
		"required": []string{"game_id", "target_id", "amount"},
	}

	desc := `Heal a target and return updated HP status.

Use when: You need to heal a character (from spell, potion, or ability).

Parameters:
  - game_id: Game session ID
  - target_id: ID of target to heal
  - amount: Amount of HP to heal

Returns: Healing result with target's new HP and status.`

	steps := []ToolStep{
		{
			ToolName: "execute_healing",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return params
			},
		},
	}

	return NewCompositeTool(
		"combat_heal",
		desc,
		schema,
		registry,
		steps,
		false,
	)
}

// NewCombatDeathSaveTool 复合死亡豁免工具：执行死亡豁免 → 获取目标状态 → 返回结果
//
// 合并: perform_death_save + get_actor
func NewCombatDeathSaveTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"game_id": map[string]any{
				"type":        "string",
				"description": "游戏会话ID",
			},
			"actor_id": map[string]any{
				"type":        "string",
				"description": "进行死亡豁免的角色ID",
			},
		},
		"required": []string{"game_id", "actor_id"},
	}

	desc := `Perform a death saving throw and return the final status (stable, dying, or dead).

Use when: A character drops to 0 HP and needs to make death saves.

Parameters:
  - game_id: Game session ID
  - actor_id: Actor making the death save

Returns: Death save result plus current HP status.`

	steps := []ToolStep{
		{
			ToolName: "perform_death_save",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return params
			},
		},
	}

	return NewCompositeTool(
		"combat_death_save",
		desc,
		schema,
		registry,
		steps,
		false,
	)
}

// NewShowCombatStatusTool 复合显示战斗状态工具：获取当前战斗 → 返回完整状态
//
// 合并: get_current_combat + get_current_turn
func NewShowCombatStatusTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"game_id": map[string]any{
				"type":        "string",
				"description": "游戏会话ID",
			},
		},
		"required": []string{"game_id"},
	}

	desc := `Show complete combat status: all participants with current HP, initiative order, and current turn.

Use when: Players ask about the current combat situation, or you need to know who is alive/dead, or what the current turn is.

Do NOT use when: Combat hasn't started yet.

Parameters:
  - game_id: Game session ID

Returns: Complete combat snapshot with all participants, HP, initiative, current turn.`

	steps := []ToolStep{
		{
			ToolName: "get_current_combat",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return params
			},
		},
	}

	return NewCompositeTool(
		"show_combat_status",
		desc,
		schema,
		registry,
		steps,
		true, // read only
	)
}

