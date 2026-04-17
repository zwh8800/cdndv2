package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// PerformAbilityCheckTool 属性检定
type PerformAbilityCheckTool struct {
	EngineTool
}

func NewPerformAbilityCheckTool(e *engine.Engine) *PerformAbilityCheckTool {
	return &PerformAbilityCheckTool{
		EngineTool: *NewEngineTool(
			"perform_ability_check",
			"执行一次属性检定",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "进行检定的角色ID",
					},
					"ability": map[string]any{
						"type":        "string",
						"enum":        []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"},
						"description": "检定的属性",
					},
					"dc": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"description": "难度等级",
					},
					"advantage": map[string]any{
						"type":        "string",
						"enum":        []string{"none", "advantage", "disadvantage"},
						"description": "优势/劣势",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "检定原因（可选）",
					},
				},
				"required": []string{"game_id", "actor_id", "ability"},
			},
			e,
			false, // write - modifies game state with check result
		),
	}
}

func (t *PerformAbilityCheckTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))
	ability := model.Ability(params["ability"].(string))

	req := engine.AbilityCheckRequest{
		GameID:  gameID,
		ActorID: actorID,
		Ability: ability,
	}

	if dc, ok := params["dc"].(float64); ok {
		req.DC = int(dc)
	}
	if adv, ok := params["advantage"].(string); ok {
		switch adv {
		case "advantage":
			req.Advantage = model.RollModifier{Advantage: true}
		case "disadvantage":
			req.Advantage = model.RollModifier{Disadvantage: true}
		}
	}
	if reason, ok := params["reason"].(string); ok {
		req.Reason = reason
	}

	result, err := e.PerformAbilityCheck(ctx, req)
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

// PerformSkillCheckTool 技能检定
type PerformSkillCheckTool struct {
	EngineTool
}

func NewPerformSkillCheckTool(e *engine.Engine) *PerformSkillCheckTool {
	return &PerformSkillCheckTool{
		EngineTool: *NewEngineTool(
			"perform_skill_check",
			"执行一次技能检定",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "进行检定的角色ID",
					},
					"skill": map[string]any{
						"type": "string",
						"enum": []string{
							"Acrobatics", "Animal Handling", "Arcana", "Athletics", "Deception",
							"History", "Insight", "Intimidation", "Investigation", "Medicine",
							"Nature", "Perception", "Performance", "Persuasion", "Religion",
							"Sleight of Hand", "Stealth", "Survival",
						},
						"description": "技能名称",
					},
					"dc": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"description": "难度等级",
					},
					"advantage": map[string]any{
						"type":        "string",
						"enum":        []string{"none", "advantage", "disadvantage"},
						"description": "优势/劣势",
					},
				},
				"required": []string{"game_id", "actor_id", "skill"},
			},
			e,
			false, // write - modifies game state with check result
		),
	}
}

func (t *PerformSkillCheckTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))
	skill := model.Skill(params["skill"].(string))

	req := engine.SkillCheckRequest{
		GameID:  gameID,
		ActorID: actorID,
		Skill:   skill,
	}

	if dc, ok := params["dc"].(float64); ok {
		req.DC = int(dc)
	}
	if adv, ok := params["advantage"].(string); ok {
		switch adv {
		case "advantage":
			req.Advantage = model.RollModifier{Advantage: true}
		case "disadvantage":
			req.Advantage = model.RollModifier{Disadvantage: true}
		}
	}

	result, err := e.PerformSkillCheck(ctx, req)
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

// PerformSavingThrowTool 豁免检定
type PerformSavingThrowTool struct {
	EngineTool
}

func NewPerformSavingThrowTool(e *engine.Engine) *PerformSavingThrowTool {
	return &PerformSavingThrowTool{
		EngineTool: *NewEngineTool(
			"perform_saving_throw",
			"执行一次豁免检定",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "进行检定的角色ID",
					},
					"ability": map[string]any{
						"type":        "string",
						"enum":        []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"},
						"description": "豁免的属性",
					},
					"dc": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"description": "难度等级",
					},
					"advantage": map[string]any{
						"type":        "string",
						"enum":        []string{"none", "advantage", "disadvantage"},
						"description": "优势/劣势",
					},
				},
				"required": []string{"game_id", "actor_id", "ability", "dc"},
			},
			e,
			false, // write - modifies game state with check result
		),
	}
}

func (t *PerformSavingThrowTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))
	ability := model.Ability(params["ability"].(string))
	dc := int(params["dc"].(float64))

	req := engine.SavingThrowRequest{
		GameID:  gameID,
		ActorID: actorID,
		Ability: ability,
		DC:      dc,
	}

	if adv, ok := params["advantage"].(string); ok {
		switch adv {
		case "advantage":
			req.Advantage = model.RollModifier{Advantage: true}
		case "disadvantage":
			req.Advantage = model.RollModifier{Disadvantage: true}
		}
	}

	result, err := e.PerformSavingThrow(ctx, req)
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

// GetPassivePerceptionTool 获取被动感知
type GetPassivePerceptionTool struct {
	EngineTool
}

func NewGetPassivePerceptionTool(e *engine.Engine) *GetPassivePerceptionTool {
	return &GetPassivePerceptionTool{
		EngineTool: *NewEngineTool(
			"get_passive_perception",
			"获取角色的被动感知值",
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
			true, // read-only - just queries a value
		),
	}
}

func (t *GetPassivePerceptionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))

	req := engine.GetPassivePerceptionRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	result, err := e.GetPassivePerception(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data: map[string]any{
			"passive_perception": result.PassivePerception,
		},
		Message: fmt.Sprintf("被动感知值: %d", result.PassivePerception),
	}, nil
}

// ShortRestTool 短休
type ShortRestTool struct {
	EngineTool
}

func NewShortRestTool(e *engine.Engine) *ShortRestTool {
	return &ShortRestTool{
		EngineTool: *NewEngineTool(
			"short_rest",
			"执行短休（至少1小时，恢复生命骰和部分HP）",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "参与短休的角色ID列表",
					},
				},
				"required": []string{"game_id", "actor_ids"},
			},
			e,
			false, // write - modifies HP/spell slots
		),
	}
}

func (t *ShortRestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))

	actorIDs := params["actor_ids"].([]any)
	aids := make([]model.ID, len(actorIDs))
	for i, aid := range actorIDs {
		aids[i] = model.ID(aid.(string))
	}

	req := engine.ShortRestRequest{
		GameID:   gameID,
		ActorIDs: aids,
	}

	result, err := e.ShortRest(ctx, req)
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

// CastSpellTool 施放法术
type CastSpellTool struct {
	EngineTool
}

func NewCastSpellTool(e *engine.Engine) *CastSpellTool {
	return &CastSpellTool{
		EngineTool: *NewEngineTool(
			"cast_spell",
			"施放一个法术",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"caster_id": map[string]any{
						"type":        "string",
						"description": "施法者ID",
					},
					"spell_id": map[string]any{
						"type":        "string",
						"description": "法术ID",
					},
					"target_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "目标ID列表",
					},
					"slot_level": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"description": "使用的法术位环级（0表示戏法）",
					},
				},
				"required": []string{"game_id", "caster_id", "spell_id"},
			},
			e,
			false, // write - uses spell slot, applies effects
		),
	}
}

func (t *CastSpellTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	casterID := model.ID(params["caster_id"].(string))
	spellID := params["spell_id"].(string)

	spell := engine.SpellInput{
		SpellID: spellID,
	}

	if sl, ok := params["slot_level"].(float64); ok {
		spell.SlotLevel = int(sl)
	}

	if tids, ok := params["target_ids"].([]any); ok {
		spell.TargetIDs = make([]model.ID, len(tids))
		for i, tid := range tids {
			spell.TargetIDs[i] = model.ID(tid.(string))
		}
	}

	req := engine.CastSpellRequest{
		GameID:   gameID,
		CasterID: casterID,
		Spell:    spell,
	}

	result, err := e.CastSpell(ctx, req)
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

// GetSpellSlotsTool 获取法术位
type GetSpellSlotsTool struct {
	EngineTool
}

func NewGetSpellSlotsTool(e *engine.Engine) *GetSpellSlotsTool {
	return &GetSpellSlotsTool{
		EngineTool: *NewEngineTool(
			"get_spell_slots",
			"获取施法者的法术位状态",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"caster_id": map[string]any{
						"type":        "string",
						"description": "施法者ID",
					},
				},
				"required": []string{"game_id", "caster_id"},
			},
			e,
			true, // read-only - just queries
		),
	}
}

func (t *GetSpellSlotsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	casterID := model.ID(params["caster_id"].(string))

	req := engine.GetSpellSlotsRequest{
		GameID:   gameID,
		CasterID: casterID,
	}

	result, err := e.GetSpellSlots(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.Info,
		Message: "法术位状态",
	}, nil
}

// PrepareSpellsTool 准备法术
type PrepareSpellsTool struct {
	EngineTool
}

func NewPrepareSpellsTool(e *engine.Engine) *PrepareSpellsTool {
	return &PrepareSpellsTool{
		EngineTool: *NewEngineTool(
			"prepare_spells",
			"准备法术（适用于准备施法者）",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"caster_id": map[string]any{
						"type":        "string",
						"description": "施法者ID",
					},
					"spell_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "要准备的法术ID列表",
					},
				},
				"required": []string{"game_id", "caster_id", "spell_ids"},
			},
			e,
			false, // write - modifies prepared spells list
		),
	}
}

func (t *PrepareSpellsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	casterID := model.ID(params["caster_id"].(string))

	spellIDs := params["spell_ids"].([]any)
	sids := make([]string, len(spellIDs))
	for i, sid := range spellIDs {
		sids[i] = sid.(string)
	}

	req := engine.PrepareSpellsRequest{
		GameID:   gameID,
		CasterID: casterID,
		SpellIDs: sids,
	}

	err := e.PrepareSpells(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: "法术准备完成",
	}, nil
}

// LearnSpellTool 学习法术
type LearnSpellTool struct {
	EngineTool
}

func NewLearnSpellTool(e *engine.Engine) *LearnSpellTool {
	return &LearnSpellTool{
		EngineTool: *NewEngineTool(
			"learn_spell",
			"学习一个新法术",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"caster_id": map[string]any{
						"type":        "string",
						"description": "施法者ID",
					},
					"spell_id": map[string]any{
						"type":        "string",
						"description": "法术ID",
					},
				},
				"required": []string{"game_id", "caster_id", "spell_id"},
			},
			e,
			false, // write - adds spell to known list
		),
	}
}

func (t *LearnSpellTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	casterID := model.ID(params["caster_id"].(string))
	spellID := params["spell_id"].(string)

	req := engine.LearnSpellRequest{
		GameID:   gameID,
		CasterID: casterID,
		SpellID:  spellID,
	}

	err := e.LearnSpell(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: "成功学习新法术",
	}, nil
}

// ConcentrationCheckTool 专注检定
type ConcentrationCheckTool struct {
	EngineTool
}

func NewConcentrationCheckTool(e *engine.Engine) *ConcentrationCheckTool {
	return &ConcentrationCheckTool{
		EngineTool: *NewEngineTool(
			"concentration_check",
			"执行专注检定",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"caster_id": map[string]any{
						"type":        "string",
						"description": "施法者ID",
					},
					"damage_taken": map[string]any{
						"type":        "integer",
						"description": "受到的伤害",
					},
				},
				"required": []string{"game_id", "caster_id", "damage_taken"},
			},
			e,
			false, // write - may break concentration
		),
	}
}

func (t *ConcentrationCheckTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	casterID := model.ID(params["caster_id"].(string))
	damageTaken := int(params["damage_taken"].(float64))

	req := engine.ConcentrationCheckRequest{
		GameID:      gameID,
		CasterID:    casterID,
		DamageTaken: damageTaken,
	}

	result, err := e.ConcentrationCheck(ctx, req)
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

// EndConcentrationTool 结束专注
type EndConcentrationTool struct {
	EngineTool
}

func NewEndConcentrationTool(e *engine.Engine) *EndConcentrationTool {
	return &EndConcentrationTool{
		EngineTool: *NewEngineTool(
			"end_concentration",
			"结束专注法术",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"caster_id": map[string]any{
						"type":        "string",
						"description": "施法者ID",
					},
				},
				"required": []string{"game_id", "caster_id"},
			},
			e,
			false, // write - removes concentration
		),
	}
}

func (t *EndConcentrationTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))
	casterID := model.ID(params["caster_id"].(string))

	req := engine.EndConcentrationRequest{
		GameID:   gameID,
		CasterID: casterID,
	}

	err := e.EndConcentration(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: "专注法术已结束",
	}, nil
}

// StartLongRestTool 开始长休
type StartLongRestTool struct {
	EngineTool
}

func NewStartLongRestTool(e *engine.Engine) *StartLongRestTool {
	return &StartLongRestTool{
		EngineTool: *NewEngineTool(
			"start_long_rest",
			"开始长休（至少8小时，完全恢复HP和法术位）",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "参与长休的角色ID列表",
					},
				},
				"required": []string{"game_id", "actor_ids"},
			},
			e,
			false, // write - starts rest
		),
	}
}

func (t *StartLongRestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))

	actorIDs := params["actor_ids"].([]any)
	aids := make([]model.ID, len(actorIDs))
	for i, aid := range actorIDs {
		aids[i] = model.ID(aid.(string))
	}

	req := engine.StartLongRestRequest{
		GameID:   gameID,
		ActorIDs: aids,
	}

	result, err := e.StartLongRest(ctx, req)
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

// EndLongRestTool 结束长休
type EndLongRestTool struct {
	EngineTool
}

func NewEndLongRestTool(e *engine.Engine) *EndLongRestTool {
	return &EndLongRestTool{
		EngineTool: *NewEngineTool(
			"end_long_rest",
			"结束长休",
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
			false, // write - restores HP/spell slots
		),
	}
}

func (t *EndLongRestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameID := model.ID(params["game_id"].(string))

	req := engine.EndLongRestRequest{
		GameID: gameID,
	}

	result, err := e.EndLongRest(ctx, req)
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
