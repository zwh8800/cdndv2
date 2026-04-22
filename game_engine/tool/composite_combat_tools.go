package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// =============================================================================
// 4. initiate_combat - 发起战斗
// =============================================================================

type InitiateCombatTool struct {
	EngineTool
}

func NewInitiateCombatTool(e *engine.Engine) *InitiateCombatTool {
	return &InitiateCombatTool{
		EngineTool: *NewEngineTool(
			"initiate_combat",
			"发起战斗遭遇。可同时创建敌人并开始战斗，自动切换到combat阶段。支持突袭战斗",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":   map[string]any{"type": "string", "description": "游戏会话ID"},
					"scene_id":  map[string]any{"type": "string", "description": "战斗发生的场景ID"},
					"participant_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "参战角色ID列表",
					},
					"enemies_to_create": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name":             map[string]any{"type": "string"},
								"hit_points":       map[string]any{"type": "integer"},
								"armor_class":      map[string]any{"type": "integer"},
								"challenge_rating": map[string]any{"type": "string"},
							},
							"required": []string{"name"},
						},
						"description": "需要同时创建的敌人列表（可选）",
					},
					"surprise": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"stealthy_side": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "潜行方角色ID"},
							"observers":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "被突袭方角色ID"},
						},
						"description": "突袭配置（可选）。提供此参数时将发起突袭战斗",
					},
				},
				"required": []string{"game_id", "scene_id", "participant_ids"},
			},
			e,
			false,
		),
	}
}

func (t *InitiateCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	sceneIDStr, err := RequireString(params, "scene_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	sceneID := model.ID(sceneIDStr)

	participantStrs, err := RequireStringArray(params, "participant_ids")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	pids := make([]model.ID, len(participantStrs))
	for i, pid := range participantStrs {
		pids[i] = model.ID(pid)
	}

	// 创建敌人（如果提供）
	var createdEnemyIDs []model.ID
	if enemiesList, ok := params["enemies_to_create"].([]any); ok {
		for _, enemyData := range enemiesList {
			enemyMap, ok := enemyData.(map[string]any)
			if !ok {
				continue
			}
			eName, _ := enemyMap["name"].(string)
			if eName == "" {
				continue
			}
			enemy := &engine.EnemyInput{
				Name:            eName,
				HitPoints:       OptionalInt(enemyMap, "hit_points", 0),
				ArmorClass:      OptionalInt(enemyMap, "armor_class", 0),
				ChallengeRating: OptionalString(enemyMap, "challenge_rating", ""),
			}
			result, createErr := e.CreateEnemy(ctx, engine.CreateEnemyRequest{GameID: gameID, Enemy: enemy})
			if createErr != nil {
				return &ToolResult{Success: false, Error: fmt.Sprintf("创建敌人 %s 失败: %v", eName, createErr)}, nil
			}
			createdEnemyIDs = append(createdEnemyIDs, result.Actor.ID)
			pids = append(pids, result.Actor.ID)
		}
	}

	// 自动切换到 combat 阶段
	_, _ = e.SetPhase(ctx, gameID, model.PhaseCombat, "战斗开始")

	// 发起战斗
	if surpriseData, ok := params["surprise"].(map[string]any); ok {
		stealthyStrs := OptionalStringArray(surpriseData, "stealthy_side")
		observerStrs := OptionalStringArray(surpriseData, "observers")
		stealthy := make([]model.ID, len(stealthyStrs))
		for i, id := range stealthyStrs {
			stealthy[i] = model.ID(id)
		}
		observers := make([]model.ID, len(observerStrs))
		for i, id := range observerStrs {
			observers[i] = model.ID(id)
		}
		result, combatErr := e.StartCombatWithSurprise(ctx, engine.StartCombatWithSurpriseRequest{
			GameID:       gameID,
			SceneID:      sceneID,
			StealthySide: stealthy,
			Observers:    observers,
		})
		if combatErr != nil {
			return &ToolResult{Success: false, Error: combatErr.Error()}, nil
		}
		return &ToolResult{
			Success: true,
			Data:    map[string]any{"combat": result.Combat, "created_enemy_ids": createdEnemyIDs},
			Message: "突袭战斗开始！",
		}, nil
	}

	result, combatErr := e.StartCombat(ctx, engine.StartCombatRequest{
		GameID:         gameID,
		SceneID:        sceneID,
		ParticipantIDs: pids,
	})
	if combatErr != nil {
		return &ToolResult{Success: false, Error: combatErr.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    map[string]any{"combat": result.Combat, "created_enemy_ids": createdEnemyIDs},
		Message: "战斗开始！先攻顺序已确定",
	}, nil
}

// =============================================================================
// 5. combat_action - 战斗动作
// =============================================================================

type CombatActionTool struct {
	EngineTool
}

func NewCombatActionTool(e *engine.Engine) *CombatActionTool {
	return &CombatActionTool{
		EngineTool: *NewEngineTool(
			"combat_action",
			"执行战斗动作。统一所有战斗操作：攻击、施法、冲刺、闪避、移动、造成伤害、治疗等",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":  map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_id": map[string]any{"type": "string", "description": "执行动作的角色ID"},
					"action": map[string]any{
						"type":        "string",
						"enum":        []string{"attack", "cast_spell", "dash", "disengage", "dodge", "help", "hide", "ready", "search", "move", "damage", "heal", "death_save"},
						"description": "动作类型",
					},
					"target_id":      map[string]any{"type": "string", "description": "目标角色ID（attack/damage/heal时必需）"},
					"weapon_id":      map[string]any{"type": "string", "description": "武器ID（attack时可选）"},
					"is_unarmed":     map[string]any{"type": "boolean", "description": "是否徒手攻击"},
					"is_off_hand":    map[string]any{"type": "boolean", "description": "是否副手攻击"},
					"spell_id":       map[string]any{"type": "string", "description": "法术ID（cast_spell时必需）"},
					"target_ids":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "法术目标ID列表"},
					"slot_level":     map[string]any{"type": "integer", "description": "法术位等级"},
					"x":              map[string]any{"type": "integer", "description": "目标X坐标（move时必需）"},
					"y":              map[string]any{"type": "integer", "description": "目标Y坐标（move时必需）"},
					"amount":         map[string]any{"type": "integer", "description": "伤害/治疗量（damage/heal时必需）"},
					"damage_type":    map[string]any{"type": "string", "description": "伤害类型（damage时可选）"},
					"advantage":      map[string]any{"type": "string", "enum": []string{"none", "advantage", "disadvantage"}, "description": "优势/劣势"},
					"auto_next_turn": map[string]any{"type": "boolean", "description": "动作完成后自动推进到下一回合（默认false）"},
				},
				"required": []string{"game_id", "actor_id", "action"},
			},
			e,
			false,
		),
	}
}

func (t *CombatActionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorID := model.ID(actorIDStr)

	action, err := RequireString(params, "action")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var result *ToolResult

	switch action {
	case "attack":
		targetIDStr, terr := RequireString(params, "target_id")
		if terr != nil {
			return &ToolResult{Success: false, Error: "attack需要target_id参数"}, nil
		}
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
		res, aerr := e.ExecuteAttack(ctx, engine.ExecuteAttackRequest{
			GameID:     gameID,
			AttackerID: actorID,
			TargetID:   model.ID(targetIDStr),
			Attack:     attack,
		})
		if aerr != nil {
			return &ToolResult{Success: false, Error: aerr.Error()}, nil
		}
		result = &ToolResult{Success: true, Data: res.AttackResult, Message: res.AttackResult.Message}

	case "cast_spell":
		spellID, serr := RequireString(params, "spell_id")
		if serr != nil {
			return &ToolResult{Success: false, Error: "cast_spell需要spell_id参数"}, nil
		}
		spell := engine.SpellInput{
			SpellID:   spellID,
			SlotLevel: OptionalInt(params, "slot_level", 0),
		}
		targetStrs := OptionalStringArray(params, "target_ids")
		if len(targetStrs) > 0 {
			spell.TargetIDs = make([]model.ID, len(targetStrs))
			for i, tid := range targetStrs {
				spell.TargetIDs[i] = model.ID(tid)
			}
		}
		res, cerr := e.CastSpell(ctx, engine.CastSpellRequest{GameID: gameID, CasterID: actorID, Spell: spell})
		if cerr != nil {
			return &ToolResult{Success: false, Error: cerr.Error()}, nil
		}
		result = &ToolResult{Success: true, Data: res, Message: res.Message}

	case "move":
		x, xerr := RequireInt(params, "x")
		if xerr != nil {
			return &ToolResult{Success: false, Error: "move需要x参数"}, nil
		}
		y, yerr := RequireInt(params, "y")
		if yerr != nil {
			return &ToolResult{Success: false, Error: "move需要y参数"}, nil
		}
		res, merr := e.MoveActor(ctx, engine.MoveActorRequest{
			GameID: gameID, ActorID: actorID, To: model.Point{X: x, Y: y},
		})
		if merr != nil {
			return &ToolResult{Success: false, Error: merr.Error()}, nil
		}
		result = &ToolResult{Success: true, Data: res, Message: res.MoveResult.Message}

	case "damage":
		targetIDStr, terr := RequireString(params, "target_id")
		if terr != nil {
			return &ToolResult{Success: false, Error: "damage需要target_id参数"}, nil
		}
		amount, aerr := RequireInt(params, "amount")
		if aerr != nil {
			return &ToolResult{Success: false, Error: "damage需要amount参数"}, nil
		}
		damage := engine.DamageInput{Amount: amount}
		if dt := OptionalString(params, "damage_type", ""); dt != "" {
			damage.Type = model.DamageType(dt)
		}
		res, derr := e.ExecuteDamage(ctx, engine.ExecuteDamageRequest{
			GameID: gameID, TargetID: model.ID(targetIDStr), Damage: damage,
		})
		if derr != nil {
			return &ToolResult{Success: false, Error: derr.Error()}, nil
		}
		result = &ToolResult{Success: true, Data: res.DamageResult, Message: res.DamageResult.Message}

	case "heal":
		targetIDStr, terr := RequireString(params, "target_id")
		if terr != nil {
			return &ToolResult{Success: false, Error: "heal需要target_id参数"}, nil
		}
		amount, aerr := RequireInt(params, "amount")
		if aerr != nil {
			return &ToolResult{Success: false, Error: "heal需要amount参数"}, nil
		}
		res, herr := e.ExecuteHealing(ctx, engine.ExecuteHealingRequest{
			GameID: gameID, TargetID: model.ID(targetIDStr), Amount: amount,
		})
		if herr != nil {
			return &ToolResult{Success: false, Error: herr.Error()}, nil
		}
		result = &ToolResult{Success: true, Data: res, Message: res.Message}

	case "death_save":
		res, derr := e.PerformSavingThrow(ctx, engine.SavingThrowRequest{
			GameID: gameID, ActorID: actorID, Ability: model.AbilityConstitution, DC: 10,
		})
		if derr != nil {
			return &ToolResult{Success: false, Error: derr.Error()}, nil
		}
		msg := fmt.Sprintf("死亡豁免: 掷骰=%d", res.RollTotal)
		if res.Success {
			msg += "，成功！"
		} else {
			msg += "，失败..."
		}
		result = &ToolResult{Success: true, Data: res, Message: msg}

	default:
		// 通用动作（dash/disengage/dodge/help/hide/ready/search）
		actionType := model.ActionType(action)
		res, aerr := e.ExecuteAction(ctx, engine.ExecuteActionRequest{
			GameID: gameID, ActorID: actorID, Action: engine.ActionInput{Type: actionType},
		})
		if aerr != nil {
			return &ToolResult{Success: false, Error: aerr.Error()}, nil
		}
		result = &ToolResult{Success: true, Data: res, Message: res.ActionResult.Message}
	}

	// 自动推进回合
	if OptionalBool(params, "auto_next_turn", false) {
		nextRes, nerr := e.NextTurn(ctx, engine.NextTurnRequest{GameID: gameID})
		if nerr == nil && nextRes != nil {
			result.Data = map[string]any{
				"action_result": result.Data,
				"next_turn":     nextRes.Combat.CurrentTurn,
			}
			result.Message += " → 下一回合: " + nextRes.Combat.CurrentTurn.ActorName
		}
	}

	return result, nil
}

// =============================================================================
// 6. resolve_combat - 结束战斗
// =============================================================================

type ResolveCombatTool struct {
	EngineTool
}

func NewResolveCombatTool(e *engine.Engine) *ResolveCombatTool {
	return &ResolveCombatTool{
		EngineTool: *NewEngineTool(
			"resolve_combat",
			"结束战斗并处理善后。可选自动分配经验值，自动切换回exploration阶段",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{"type": "string", "description": "游戏会话ID"},
					"xp_awards": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"pc_id": map[string]any{"type": "string", "description": "PC角色ID"},
								"xp":    map[string]any{"type": "integer", "description": "经验值"},
							},
							"required": []string{"pc_id", "xp"},
						},
						"description": "经验分配列表（可选）",
					},
				},
				"required": []string{"game_id"},
			},
			e,
			false,
		),
	}
}

func (t *ResolveCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	// 结束战斗
	endErr := e.EndCombat(ctx, engine.EndCombatRequest{GameID: gameID})
	if endErr != nil {
		return &ToolResult{Success: false, Error: endErr.Error()}, nil
	}

	// 分配经验
	var xpResults []map[string]any
	if xpList, ok := params["xp_awards"].([]any); ok {
		for _, xpData := range xpList {
			xpMap, ok := xpData.(map[string]any)
			if !ok {
				continue
			}
			pcIDStr, _ := xpMap["pc_id"].(string)
			xpVal, _ := xpMap["xp"].(float64)
			if pcIDStr == "" || xpVal <= 0 {
				continue
			}
			res, xpErr := e.AddExperience(ctx, engine.AddExperienceRequest{
				GameID: gameID,
				PCID:   model.ID(pcIDStr),
				XP:     int(xpVal),
			})
			if xpErr == nil {
				xpResults = append(xpResults, map[string]any{
					"pc_id":    pcIDStr,
					"xp_added": int(xpVal),
					"leveled_up": res.LeveledUp,
				})
			}
		}
	}

	// 自动切换回 exploration 阶段
	_, _ = e.SetPhase(ctx, gameID, model.PhaseExploration, "战斗结束，回到探索阶段")

	return &ToolResult{
		Success: true,
		Data:    map[string]any{"xp_awards": xpResults},
		Message: "战斗结束，已切换到探索阶段",
	}, nil
}

// =============================================================================
// 7. query_combat - 查询战斗状态
// =============================================================================

type QueryCombatTool struct {
	EngineTool
}

func NewQueryCombatTool(e *engine.Engine) *QueryCombatTool {
	return &QueryCombatTool{
		EngineTool: *NewEngineTool(
			"query_combat",
			"查询当前战斗状态和回合信息",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":            map[string]any{"type": "string", "description": "游戏会话ID"},
					"include_turn_info": map[string]any{"type": "boolean", "description": "是否包含当前回合详情（默认true）"},
				},
				"required": []string{"game_id"},
			},
			e,
			true, // 只读
		),
	}
}

func (t *QueryCombatTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	combatResult, err := e.GetCurrentCombat(ctx, engine.GetCurrentCombatRequest{GameID: gameID})
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	data := map[string]any{"combat": combatResult.Combat}

	if OptionalBool(params, "include_turn_info", true) {
		turnResult, terr := e.GetCurrentTurn(ctx, engine.GetCurrentTurnRequest{GameID: gameID})
		if terr == nil {
			data["current_turn"] = turnResult
		}
	}

	return &ToolResult{
		Success: true,
		Data:    data,
		Message: "当前战斗状态",
	}, nil
}
