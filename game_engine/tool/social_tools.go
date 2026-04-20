package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 社交互动工具 ==========

// InteractWithNPCTool 与NPC互动
type InteractWithNPCTool struct {
	EngineTool
}

func NewInteractWithNPCTool(e *engine.Engine) *InteractWithNPCTool {
	return &InteractWithNPCTool{
		EngineTool: *NewEngineTool(
			"interact_with_npc",
			"与NPC进行社交互动（游说、威吓、欺瞒等），根据检定结果更新NPC态度",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"npc_id": map[string]any{
						"type":        "string",
						"description": "NPC ID",
					},
					"check_type": map[string]any{
						"type":        "string",
						"description": "社交检定类型 (persuasion=游说, intimidation=威吓, deception=欺瞒, performance=表演)",
					},
					"ability": map[string]any{
						"type":        "integer",
						"description": "相关属性值",
					},
					"prof_bonus": map[string]any{
						"type":        "integer",
						"description": "熟练加值",
					},
					"has_proficiency": map[string]any{
						"type":        "boolean",
						"description": "是否有技能熟练",
					},
				},
				"required": []string{"game_id", "npc_id", "check_type", "ability"},
			},
			e,
			false,
		),
	}
}

func (t *InteractWithNPCTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	npcIDStr, err := RequireString(params, "npc_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	npcID := model.ID(npcIDStr)

	checkTypeStr, err := RequireString(params, "check_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	checkType := model.SocialCheckType(checkTypeStr)

	ability, err := RequireInt(params, "ability")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	profBonus := OptionalInt(params, "prof_bonus", 0)
	hasProf := OptionalBool(params, "has_proficiency", false)

	req := engine.InteractWithNPCRequest{
		GameID:    gameID,
		NPCID:     npcID,
		CheckType: checkType,
		Ability:   ability,
		ProfBonus: profBonus,
		HasProf:   hasProf,
	}

	result, err := e.InteractWithNPC(ctx, req)
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

// GetNPCAttitudeTool 获取NPC态度
type GetNPCAttitudeTool struct {
	EngineTool
}

func NewGetNPCAttitudeTool(e *engine.Engine) *GetNPCAttitudeTool {
	return &GetNPCAttitudeTool{
		EngineTool: *NewEngineTool(
			"get_npc_attitude",
			"获取NPC当前对玩家的态度",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"npc_id": map[string]any{
						"type":        "string",
						"description": "NPC ID",
					},
				},
				"required": []string{"game_id", "npc_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetNPCAttitudeTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	npcIDStr, err := RequireString(params, "npc_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	npcID := model.ID(npcIDStr)

	req := engine.GetNPCAttitudeRequest{
		GameID: gameID,
		NPCID:  npcID,
	}

	result, err := e.GetNPCAttitude(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("NPC态度: %s, 性格: %s", result.Attitude, result.Disposition),
	}, nil
}
