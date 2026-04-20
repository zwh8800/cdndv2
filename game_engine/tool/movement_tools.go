package tool

import (
	"context"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 移动工具 ==========

// PerformJumpTool 执行跳跃
type PerformJumpTool struct {
	EngineTool
}

func NewPerformJumpTool(e *engine.Engine) *PerformJumpTool {
	return &PerformJumpTool{
		EngineTool: *NewEngineTool(
			"perform_jump",
			"执行跳跃动作（跳远或跳高），根据力量值计算距离",
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
					"jump_type": map[string]any{
						"type":        "string",
						"description": "跳跃类型 (long=跳远, high=跳高)",
					},
					"has_running_start": map[string]any{
						"type":        "boolean",
						"description": "是否有助跑（默认false）",
					},
				},
				"required": []string{"game_id", "actor_id", "jump_type"},
			},
			e,
			false,
		),
	}
}

func (t *PerformJumpTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	jumpTypeStr, err := RequireString(params, "jump_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	jumpType := model.JumpType(jumpTypeStr)

	hasRunningStart := OptionalBool(params, "has_running_start", false)

	req := engine.PerformJumpRequest{
		GameID:          gameID,
		ActorID:         actorID,
		JumpType:        jumpType,
		HasRunningStart: hasRunningStart,
	}

	result, err := e.PerformJump(ctx, req)
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

// ApplyFallDamageTool 应用跌落伤害
type ApplyFallDamageTool struct {
	EngineTool
}

func NewApplyFallDamageTool(e *engine.Engine) *ApplyFallDamageTool {
	return &ApplyFallDamageTool{
		EngineTool: *NewEngineTool(
			"apply_fall_damage",
			"应用跌落伤害（每10尺1d6，最多20d6）",
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
					"fall_distance": map[string]any{
						"type":        "integer",
						"description": "跌落距离（尺）",
					},
				},
				"required": []string{"game_id", "actor_id", "fall_distance"},
			},
			e,
			false,
		),
	}
}

func (t *ApplyFallDamageTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	fallDistance, err := RequireInt(params, "fall_distance")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.ApplyFallDamageRequest{
		GameID:       gameID,
		ActorID:      actorID,
		FallDistance: fallDistance,
	}

	result, err := e.ApplyFallDamage(ctx, req)
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

// CalculateBreathHoldingTool 计算闭气能力
type CalculateBreathHoldingTool struct {
	EngineTool
}

func NewCalculateBreathHoldingTool(e *engine.Engine) *CalculateBreathHoldingTool {
	return &CalculateBreathHoldingTool{
		EngineTool: *NewEngineTool(
			"calculate_breath_holding",
			"计算角色闭气能力",
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
			true,
		),
	}
}

func (t *CalculateBreathHoldingTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.CalculateBreathHoldingRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	result, err := e.CalculateBreathHolding(ctx, req)
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

// ApplySuffocationTool 应用窒息效果
type ApplySuffocationTool struct {
	EngineTool
}

func NewApplySuffocationTool(e *engine.Engine) *ApplySuffocationTool {
	return &ApplySuffocationTool{
		EngineTool: *NewEngineTool(
			"apply_suffocation",
			"应用窒息效果",
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

func (t *ApplySuffocationTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.ApplySuffocationRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	result, err := e.ApplySuffocation(ctx, req)
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

// PerformEncounterCheckTool 执行遭遇检定
type PerformEncounterCheckTool struct {
	EngineTool
}

func NewPerformEncounterCheckTool(e *engine.Engine) *PerformEncounterCheckTool {
	return &PerformEncounterCheckTool{
		EngineTool: *NewEngineTool(
			"perform_encounter_check",
			"执行随机遭遇检定（1d6，1-2发生遭遇）",
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

func (t *PerformEncounterCheckTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	req := engine.PerformEncounterCheckRequest{
		GameID: gameID,
	}

	result, err := e.PerformEncounterCheck(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: result.Encountered,
		Data:    result,
		Message: result.Message,
	}, nil
}
