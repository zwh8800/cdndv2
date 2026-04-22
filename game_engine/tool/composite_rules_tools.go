package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// =============================================================================
// 8. perform_check - 统一检定
// =============================================================================

type PerformCheckTool struct {
	EngineTool
}

func NewPerformCheckTool(e *engine.Engine) *PerformCheckTool {
	return &PerformCheckTool{
		EngineTool: *NewEngineTool(
			"perform_check",
			"执行D&D检定：属性检定、技能检定或豁免检定。统一入口，通过check_type区分",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":  map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_id": map[string]any{"type": "string", "description": "进行检定的角色ID"},
					"check_type": map[string]any{
						"type": "string", "enum": []string{"ability", "skill", "saving_throw"},
						"description": "检定类型",
					},
					"ability": map[string]any{
						"type": "string",
						"enum": []string{"strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma"},
						"description": "属性（ability/saving_throw时必需）",
					},
					"skill": map[string]any{
						"type":        "string",
						"description": "技能名称（skill时必需），如athletics/acrobatics/perception/stealth等",
					},
					"dc":        map[string]any{"type": "integer", "description": "难度等级（可选，saving_throw时必需）"},
					"advantage": map[string]any{"type": "string", "enum": []string{"none", "advantage", "disadvantage"}, "description": "优势/劣势"},
					"reason":    map[string]any{"type": "string", "description": "检定原因描述"},
				},
				"required": []string{"game_id", "actor_id", "check_type"},
			},
			e,
			false,
		),
	}
}

func (t *PerformCheckTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	checkType, err := RequireString(params, "check_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	adv := OptionalString(params, "advantage", "none")
	var rollMod model.RollModifier
	switch adv {
	case "advantage":
		rollMod = model.RollModifier{Advantage: true}
	case "disadvantage":
		rollMod = model.RollModifier{Disadvantage: true}
	}

	switch checkType {
	case "ability":
		abilityStr, aerr := RequireString(params, "ability")
		if aerr != nil {
			return &ToolResult{Success: false, Error: "ability检定需要ability参数"}, nil
		}
		result, cerr := e.PerformAbilityCheck(ctx, engine.AbilityCheckRequest{
			GameID:    gameID,
			ActorID:   actorID,
			Ability:   model.Ability(abilityStr),
			DC:        OptionalInt(params, "dc", 0),
			Reason:    OptionalString(params, "reason", ""),
			Advantage: rollMod,
		})
		if cerr != nil {
			return &ToolResult{Success: false, Error: cerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	case "skill":
		skillStr, serr := RequireString(params, "skill")
		if serr != nil {
			return &ToolResult{Success: false, Error: "skill检定需要skill参数"}, nil
		}
		result, cerr := e.PerformSkillCheck(ctx, engine.SkillCheckRequest{
			GameID:    gameID,
			ActorID:   actorID,
			Skill:     model.Skill(skillStr),
			DC:        OptionalInt(params, "dc", 0),
			Advantage: rollMod,
		})
		if cerr != nil {
			return &ToolResult{Success: false, Error: cerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	case "saving_throw":
		abilityStr, aerr := RequireString(params, "ability")
		if aerr != nil {
			return &ToolResult{Success: false, Error: "saving_throw需要ability参数"}, nil
		}
		dc, derr := RequireInt(params, "dc")
		if derr != nil {
			return &ToolResult{Success: false, Error: "saving_throw需要dc参数"}, nil
		}
		result, cerr := e.PerformSavingThrow(ctx, engine.SavingThrowRequest{
			GameID:    gameID,
			ActorID:   actorID,
			Ability:   model.Ability(abilityStr),
			DC:        dc,
			Advantage: rollMod,
		})
		if cerr != nil {
			return &ToolResult{Success: false, Error: cerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效的check_type: %s", checkType)}, nil
	}
}

// =============================================================================
// 9. cast_spell - 施放法术（组合工具，自动处理专注）
// =============================================================================

type CastSpellCompositeTool struct {
	EngineTool
}

func NewCastSpellCompositeTool(e *engine.Engine) *CastSpellCompositeTool {
	return &CastSpellCompositeTool{
		EngineTool: *NewEngineTool(
			"cast_spell",
			"施放法术。自动消耗法术位，处理专注法术冲突",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":    map[string]any{"type": "string", "description": "游戏会话ID"},
					"caster_id":  map[string]any{"type": "string", "description": "施法者角色ID"},
					"spell_id":   map[string]any{"type": "string", "description": "法术ID"},
					"target_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "目标ID列表（可选）"},
					"slot_level": map[string]any{"type": "integer", "description": "使用的法术位等级（可选，升环施法时使用）"},
				},
				"required": []string{"game_id", "caster_id", "spell_id"},
			},
			e,
			false,
		),
	}
}

func (t *CastSpellCompositeTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	casterIDStr, err := RequireString(params, "caster_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	casterID := model.ID(casterIDStr)

	spellID, err := RequireString(params, "spell_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
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

	result, castErr := e.CastSpell(ctx, engine.CastSpellRequest{
		GameID:   gameID,
		CasterID: casterID,
		Spell:    spell,
	})
	if castErr != nil {
		return &ToolResult{Success: false, Error: castErr.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.Message,
	}, nil
}

// =============================================================================
// 10. manage_spells - 法术管理
// =============================================================================

type ManageSpellsTool struct {
	EngineTool
}

func NewManageSpellsTool(e *engine.Engine) *ManageSpellsTool {
	return &ManageSpellsTool{
		EngineTool: *NewEngineTool(
			"manage_spells",
			"管理法术：学习新法术或准备每日法术列表",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":   map[string]any{"type": "string", "description": "游戏会话ID"},
					"caster_id": map[string]any{"type": "string", "description": "施法者角色ID"},
					"operation": map[string]any{
						"type": "string", "enum": []string{"learn", "prepare"},
						"description": "操作类型：learn=学习法术，prepare=准备法术列表",
					},
					"spell_id":  map[string]any{"type": "string", "description": "法术ID（learn时必需）"},
					"spell_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "法术ID列表（prepare时必需）"},
				},
				"required": []string{"game_id", "caster_id", "operation"},
			},
			e,
			false,
		),
	}
}

func (t *ManageSpellsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	casterIDStr, err := RequireString(params, "caster_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	casterID := model.ID(casterIDStr)

	op, err := RequireString(params, "operation")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	switch op {
	case "learn":
		spellID, serr := RequireString(params, "spell_id")
		if serr != nil {
			return &ToolResult{Success: false, Error: "learn操作需要spell_id参数"}, nil
		}
		lerr := e.LearnSpell(ctx, engine.LearnSpellRequest{GameID: gameID, CasterID: casterID, SpellID: spellID})
		if lerr != nil {
			return &ToolResult{Success: false, Error: lerr.Error()}, nil
		}
		return &ToolResult{Success: true, Message: "成功学习新法术"}, nil

	case "prepare":
		spellIDs, serr := RequireStringArray(params, "spell_ids")
		if serr != nil {
			return &ToolResult{Success: false, Error: "prepare操作需要spell_ids参数"}, nil
		}
		perr := e.PrepareSpells(ctx, engine.PrepareSpellsRequest{GameID: gameID, CasterID: casterID, SpellIDs: spellIDs})
		if perr != nil {
			return &ToolResult{Success: false, Error: perr.Error()}, nil
		}
		return &ToolResult{Success: true, Message: fmt.Sprintf("成功准备 %d 个法术", len(spellIDs))}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效操作: %s", op)}, nil
	}
}

// =============================================================================
// 11. take_rest - 统一休息
// =============================================================================

type TakeRestTool struct {
	EngineTool
}

func NewTakeRestTool(e *engine.Engine) *TakeRestTool {
	return &TakeRestTool{
		EngineTool: *NewEngineTool(
			"take_rest",
			"进行短休或长休。自动处理HP恢复、法术位恢复、魔法物品充能等",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_ids": map[string]any{
						"type": "array", "items": map[string]any{"type": "string"},
						"description": "参与休息的角色ID列表",
					},
					"rest_type": map[string]any{
						"type": "string", "enum": []string{"short", "long"},
						"description": "休息类型",
					},
				},
				"required": []string{"game_id", "actor_ids", "rest_type"},
			},
			e,
			false,
		),
	}
}

func (t *TakeRestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	actorStrs, err := RequireStringArray(params, "actor_ids")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	aids := make([]model.ID, len(actorStrs))
	for i, aid := range actorStrs {
		aids[i] = model.ID(aid)
	}

	restType, err := RequireString(params, "rest_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	switch restType {
	case "short":
		result, rerr := e.ShortRest(ctx, engine.ShortRestRequest{GameID: gameID, ActorIDs: aids})
		if rerr != nil {
			return &ToolResult{Success: false, Error: rerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	case "long":
		startResult, serr := e.StartLongRest(ctx, engine.StartLongRestRequest{GameID: gameID, ActorIDs: aids})
		if serr != nil {
			return &ToolResult{Success: false, Error: serr.Error()}, nil
		}
		endResult, eerr := e.EndLongRest(ctx, engine.EndLongRestRequest{GameID: gameID})
		if eerr != nil {
			return &ToolResult{Success: false, Error: eerr.Error()}, nil
		}
		// 自动恢复魔法物品充能
		for _, aid := range aids {
			_, _ = e.RechargeMagicItems(ctx, engine.RechargeMagicItemsRequest{GameID: gameID, ActorID: aid})
		}
		return &ToolResult{
			Success: true,
			Data:    map[string]any{"start": startResult, "end": endResult},
			Message: endResult.Message,
		}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效的rest_type: %s", restType)}, nil
	}
}

// =============================================================================
// 12. query_spell_status - 法术状态查询
// =============================================================================

type QuerySpellStatusTool struct {
	EngineTool
}

func NewQuerySpellStatusTool(e *engine.Engine) *QuerySpellStatusTool {
	return &QuerySpellStatusTool{
		EngineTool: *NewEngineTool(
			"query_spell_status",
			"查询角色的法术位状态，可选包含被动感知值",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":                      map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_id":                     map[string]any{"type": "string", "description": "角色ID"},
					"include_passive_perception": map[string]any{"type": "boolean", "description": "是否包含被动感知（默认false）"},
				},
				"required": []string{"game_id", "actor_id"},
			},
			e,
			true, // 只读
		),
	}
}

func (t *QuerySpellStatusTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	data := map[string]any{}

	slotsResult, serr := e.GetSpellSlots(ctx, engine.GetSpellSlotsRequest{GameID: gameID, CasterID: actorID})
	if serr != nil {
		return &ToolResult{Success: false, Error: serr.Error()}, nil
	}
	data["spell_slots"] = slotsResult.Info

	if OptionalBool(params, "include_passive_perception", false) {
		ppResult, perr := e.GetPassivePerception(ctx, engine.GetPassivePerceptionRequest{GameID: gameID, ActorID: actorID})
		if perr == nil {
			data["passive_perception"] = ppResult.PassivePerception
		}
	}

	return &ToolResult{
		Success: true,
		Data:    data,
		Message: "法术状态查询完成",
	}, nil
}
