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
		),
	}
}

func (t *StartCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	sceneID := model.ID(params["scene_id"].(string))

	participantIDs := params["participant_ids"].([]any)
	pids := make([]model.ID, len(participantIDs))
	for i, pid := range participantIDs {
		pids[i] = model.ID(pid.(string))
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
		),
	}
}

func (t *StartCombatWithSurpriseTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	sceneID := model.ID(params["scene_id"].(string))

	stealthyIDs := params["stealthy_side"].([]any)
	stealthy := make([]model.ID, len(stealthyIDs))
	for i, id := range stealthyIDs {
		stealthy[i] = model.ID(id.(string))
	}

	observerIDs := params["observers"].([]any)
	observers := make([]model.ID, len(observerIDs))
	for i, id := range observerIDs {
		observers[i] = model.ID(id.(string))
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
			"获取当前战斗状态",
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
		),
	}
}

func (t *GetCurrentCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))

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
			"获取当前回合信息",
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
		),
	}
}

func (t *GetCurrentTurnTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))

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
		),
	}
}

func (t *NextTurnTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))

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
		),
	}
}

func (t *ExecuteActionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))
	actionType := model.ActionType(params["action_type"].(string))

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
		),
	}
}

func (t *ExecuteAttackTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	attackerID := model.ID(params["attacker_id"].(string))
	targetID := model.ID(params["target_id"].(string))

	attack := engine.AttackInput{}

	if wid, ok := params["weapon_id"].(string); ok && wid != "" {
		id := model.ID(wid)
		attack.WeaponID = &id
	}
	if unarmed, ok := params["is_unarmed"].(bool); ok {
		attack.IsUnarmed = unarmed
	}
	if offHand, ok := params["is_off_hand"].(bool); ok {
		attack.IsOffHand = offHand
	}
	if adv, ok := params["advantage"].(string); ok {
		switch adv {
		case "advantage":
			attack.Advantage = model.RollModifier{Advantage: true}
		case "disadvantage":
			attack.Advantage = model.RollModifier{Disadvantage: true}
		}
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
		),
	}
}

func (t *MoveActorTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))
	x := int(params["x"].(float64))
	y := int(params["y"].(float64))

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
		),
	}
}

func (t *ExecuteDamageTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	targetID := model.ID(params["target_id"].(string))
	amount := int(params["amount"].(float64))

	damage := engine.DamageInput{
		Amount: amount,
	}

	if dt, ok := params["damage_type"].(string); ok {
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
		),
	}
}

func (t *ExecuteHealingTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	targetID := model.ID(params["target_id"].(string))
	amount := int(params["amount"].(float64))

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
		),
	}
}

func (t *PerformDeathSaveTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))

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
		),
	}
}

func (t *EndCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))

	req := engine.EndCombatRequest{
		GameID: gameID,
	}

	err := e.EndCombat(ctx, req)
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
