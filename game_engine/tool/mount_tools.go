package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 坐骑工具 ==========

// MountCreatureTool 骑上坐骑
type MountCreatureTool struct {
	EngineTool
}

func NewMountCreatureTool(e *engine.Engine) *MountCreatureTool {
	return &MountCreatureTool{
		EngineTool: *NewEngineTool(
			"mount_creature",
			"骑上指定坐骑",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"rider_id": map[string]any{
						"type":        "string",
						"description": "骑手ID",
					},
					"mount_id": map[string]any{
						"type":        "string",
						"description": "坐骑ID",
					},
				},
				"required": []string{"game_id", "rider_id", "mount_id"},
			},
			e,
			false,
		),
	}
}

func (t *MountCreatureTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	riderIDStr, err := RequireString(params, "rider_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	riderID := model.ID(riderIDStr)

	mountIDStr, err := RequireString(params, "mount_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	mountID := model.ID(mountIDStr)

	req := engine.MountCreatureRequest{
		GameID:  gameID,
		RiderID: riderID,
		MountID: mountID,
	}

	result, err := e.MountCreature(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: result.Success,
		Data:    result,
		Message: result.Message,
	}, nil
}

// DismountTool 下马
type DismountTool struct {
	EngineTool
}

func NewDismountTool(e *engine.Engine) *DismountTool {
	return &DismountTool{
		EngineTool: *NewEngineTool(
			"dismount",
			"从坐骑上下来",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"rider_id": map[string]any{
						"type":        "string",
						"description": "骑手ID",
					},
				},
				"required": []string{"game_id", "rider_id"},
			},
			e,
			false,
		),
	}
}

func (t *DismountTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	riderIDStr, err := RequireString(params, "rider_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	riderID := model.ID(riderIDStr)

	req := engine.DismountRequest{
		GameID:  gameID,
		RiderID: riderID,
	}

	result, err := e.Dismount(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: result.Success,
		Data:    result,
		Message: result.Message,
	}, nil
}

// CalculateMountSpeedTool 计算坐骑速度
type CalculateMountSpeedTool struct {
	EngineTool
}

func NewCalculateMountSpeedTool(e *engine.Engine) *CalculateMountSpeedTool {
	return &CalculateMountSpeedTool{
		EngineTool: *NewEngineTool(
			"calculate_mount_speed",
			"计算坐骑移动速度和载重能力",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"mount_id": map[string]any{
						"type":        "string",
						"description": "坐骑ID（可以是数据ID或角色ID）",
					},
				},
				"required": []string{"game_id", "mount_id"},
			},
			e,
			true,
		),
	}
}

func (t *CalculateMountSpeedTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	mountIDStr, err := RequireString(params, "mount_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	mountID := model.ID(mountIDStr)

	req := engine.CalculateMountSpeedRequest{
		GameID:  gameID,
		MountID: mountID,
	}

	result, err := e.CalculateMountSpeed(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("坐骑速度: %d尺, 载重: %.1f磅", result.FinalSpeed, result.CarryCap),
	}, nil
}
