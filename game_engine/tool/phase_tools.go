package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 游戏阶段管理工具 ==========

// SetPhaseTool 设置游戏阶段
type SetPhaseTool struct {
	EngineTool
}

func NewSetPhaseTool(e *engine.Engine) *SetPhaseTool {
	return &SetPhaseTool{
		EngineTool: *NewEngineTool(
			"set_phase",
			"切换游戏阶段。当角色创建完成后应切换到 exploration，战斗开始时切换到 combat，休息时切换到 rest。可用阶段: character_creation, exploration, combat, rest",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"phase": map[string]any{
						"type":        "string",
						"description": "目标游戏阶段。可选值: character_creation, exploration, combat, rest",
						"enum":        []string{"character_creation", "exploration", "combat", "rest"},
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "阶段转换的原因说明",
					},
				},
				"required": []string{"game_id", "phase", "reason"},
			},
			e,
			false,
		),
	}
}

func (t *SetPhaseTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	phaseStr, err := RequireString(params, "phase")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	phase := model.Phase(phaseStr)

	// 验证阶段值
	validPhases := map[model.Phase]bool{
		model.PhaseCharacterCreation: true,
		model.PhaseExploration:       true,
		model.PhaseCombat:            true,
		model.PhaseRest:              true,
	}
	if !validPhases[phase] {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid phase: %s, valid phases: character_creation, exploration, combat, rest", phaseStr),
		}, nil
	}

	reason, err := RequireString(params, "reason")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	result, err := e.SetPhase(ctx, gameID, phase, reason)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("游戏阶段已从 %s 切换到 %s: %s", result.OldPhase, result.NewPhase, result.Message),
	}, nil
}

// GetPhaseTool 获取当前游戏阶段
type GetPhaseTool struct {
	EngineTool
}

func NewGetPhaseTool(e *engine.Engine) *GetPhaseTool {
	return &GetPhaseTool{
		EngineTool: *NewEngineTool(
			"get_phase",
			"获取当前游戏阶段",
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

func (t *GetPhaseTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	phase, err := e.GetPhase(ctx, gameID)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    map[string]string{"phase": string(phase)},
		Message: fmt.Sprintf("当前游戏阶段: %s", phase),
	}, nil
}
