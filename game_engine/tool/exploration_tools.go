package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 探索工具 ==========

// StartTravelTool 开始旅行
type StartTravelTool struct {
	EngineTool
}

func NewStartTravelTool(e *engine.Engine) *StartTravelTool {
	return &StartTravelTool{
		EngineTool: *NewEngineTool(
			"start_travel",
			"开始一段新的旅行",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"destination": map[string]any{
						"type":        "string",
						"description": "目的地名称",
					},
					"pace": map[string]any{
						"type":        "string",
						"description": "行进速度 (fast=快速, normal=正常, slow=慢速)",
					},
					"terrain": map[string]any{
						"type":        "string",
						"description": "地形类型 (clear, grassland, forest, mountain, swamp, desert, arctic)",
					},
					"distance": map[string]any{
						"type":        "number",
						"description": "总距离（英里）",
					},
				},
				"required": []string{"game_id", "destination", "pace", "terrain", "distance"},
			},
			e,
			false,
		),
	}
}

func (t *StartTravelTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	destination, err := RequireString(params, "destination")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	paceStr, err := RequireString(params, "pace")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	pace := model.TravelPace(paceStr)

	terrainStr, err := RequireString(params, "terrain")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	terrain := model.TerrainType(terrainStr)

	distance, err := RequireFloat(params, "distance")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.StartTravelRequest{
		GameID:      gameID,
		Destination: destination,
		Pace:        pace,
		Terrain:     terrain,
		Distance:    distance,
	}

	result, err := e.StartTravel(ctx, req)
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

// AdvanceTravelTool 推进旅行
type AdvanceTravelTool struct {
	EngineTool
}

func NewAdvanceTravelTool(e *engine.Engine) *AdvanceTravelTool {
	return &AdvanceTravelTool{
		EngineTool: *NewEngineTool(
			"advance_travel",
			"推进旅行进度（按小时计算）",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"hours": map[string]any{
						"type":        "integer",
						"description": "行进小时数",
					},
				},
				"required": []string{"game_id", "hours"},
			},
			e,
			false,
		),
	}
}

func (t *AdvanceTravelTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	hours, err := RequireInt(params, "hours")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.AdvanceTravelRequest{
		GameID: gameID,
		Hours:  hours,
	}

	result, err := e.AdvanceTravel(ctx, req)
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

// ForageTool 觅食
type ForageTool struct {
	EngineTool
}

func NewForageTool(e *engine.Engine) *ForageTool {
	return &ForageTool{
		EngineTool: *NewEngineTool(
			"forage",
			"在野外环境中觅食寻找食物和水源",
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

func (t *ForageTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	req := engine.ForageRequest{
		GameID: gameID,
	}

	result, err := e.Forage(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("觅食结果: 成功=%v", result.Result.Success),
	}, nil
}

// NavigateTool 导航检定
type NavigateTool struct {
	EngineTool
}

func NewNavigateTool(e *engine.Engine) *NavigateTool {
	return &NavigateTool{
		EngineTool: *NewEngineTool(
			"navigate",
			"进行导航检定以确定方向",
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

func (t *NavigateTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	req := engine.NavigateRequest{
		GameID: gameID,
	}

	result, err := e.Navigate(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	msg := "导航成功"
	if result.Result.Lost {
		msg = "导航失败，迷失方向"
	}

	return &ToolResult{
		Success: !result.Result.Lost,
		Data:    result,
		Message: msg,
	}, nil
}

// ========== 陷阱工具 ==========

// PlaceTrapTool 放置陷阱
type PlaceTrapTool struct {
	EngineTool
}

func NewPlaceTrapTool(e *engine.Engine) *PlaceTrapTool {
	return &PlaceTrapTool{
		EngineTool: *NewEngineTool(
			"place_trap",
			"在指定场景位置放置陷阱",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"scene_id": map[string]any{
						"type":        "string",
						"description": "场景ID",
					},
					"trap_id": map[string]any{
						"type":        "string",
						"description": "陷阱数据ID",
					},
					"position": map[string]any{
						"type":        "string",
						"description": "陷阱放置位置描述",
					},
				},
				"required": []string{"game_id", "scene_id", "trap_id", "position"},
			},
			e,
			false,
		),
	}
}

func (t *PlaceTrapTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	trapID, err := RequireString(params, "trap_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	position, err := RequireString(params, "position")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.PlaceTrapRequest{
		GameID:   gameID,
		SceneID:  sceneID,
		TrapID:   trapID,
		Position: position,
	}

	result, err := e.PlaceTrap(ctx, req)
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

// DetectTrapTool 检测陷阱
type DetectTrapTool struct {
	EngineTool
}

func NewDetectTrapTool(e *engine.Engine) *DetectTrapTool {
	return &DetectTrapTool{
		EngineTool: *NewEngineTool(
			"detect_trap",
			"检测场景中的陷阱（需要进行感知察觉检定）",
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
					"scene_id": map[string]any{
						"type":        "string",
						"description": "场景ID",
					},
					"trap_id": map[string]any{
						"type":        "string",
						"description": "陷阱ID",
					},
				},
				"required": []string{"game_id", "actor_id", "scene_id", "trap_id"},
			},
			e,
			false,
		),
	}
}

func (t *DetectTrapTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	sceneIDStr, err := RequireString(params, "scene_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	sceneID := model.ID(sceneIDStr)

	trapIDStr, err := RequireString(params, "trap_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	trapID := model.ID(trapIDStr)

	req := engine.DetectTrapRequest{
		GameID:  gameID,
		ActorID: actorID,
		SceneID: sceneID,
		TrapID:  trapID,
	}

	result, err := e.DetectTrap(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: result.TrapRevealed,
		Data:    result,
		Message: result.Message,
	}, nil
}

// DisarmTrapTool 解除陷阱
type DisarmTrapTool struct {
	EngineTool
}

func NewDisarmTrapTool(e *engine.Engine) *DisarmTrapTool {
	return &DisarmTrapTool{
		EngineTool: *NewEngineTool(
			"disarm_trap",
			"尝试解除场景中的陷阱（需要进行敏捷巧手检定）",
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
					"scene_id": map[string]any{
						"type":        "string",
						"description": "场景ID",
					},
					"trap_id": map[string]any{
						"type":        "string",
						"description": "陷阱ID",
					},
				},
				"required": []string{"game_id", "actor_id", "scene_id", "trap_id"},
			},
			e,
			false,
		),
	}
}

func (t *DisarmTrapTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	sceneIDStr, err := RequireString(params, "scene_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	sceneID := model.ID(sceneIDStr)

	trapIDStr, err := RequireString(params, "trap_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	trapID := model.ID(trapIDStr)

	req := engine.DisarmTrapRequest{
		GameID:  gameID,
		ActorID: actorID,
		SceneID: sceneID,
		TrapID:  trapID,
	}

	result, err := e.DisarmTrap(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: result.TrapDisarmed,
		Data:    result,
		Message: result.Message,
	}, nil
}

// TriggerTrapTool 触发陷阱
type TriggerTrapTool struct {
	EngineTool
}

func NewTriggerTrapTool(e *engine.Engine) *TriggerTrapTool {
	return &TriggerTrapTool{
		EngineTool: *NewEngineTool(
			"trigger_trap",
			"触发陷阱并应用其效果",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "触发陷阱的角色ID",
					},
					"scene_id": map[string]any{
						"type":        "string",
						"description": "场景ID",
					},
					"trap_id": map[string]any{
						"type":        "string",
						"description": "陷阱ID",
					},
				},
				"required": []string{"game_id", "actor_id", "scene_id", "trap_id"},
			},
			e,
			false,
		),
	}
}

func (t *TriggerTrapTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	sceneIDStr, err := RequireString(params, "scene_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	sceneID := model.ID(sceneIDStr)

	trapIDStr, err := RequireString(params, "trap_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	trapID := model.ID(trapIDStr)

	req := engine.TriggerTrapRequest{
		GameID:  gameID,
		ActorID: actorID,
		SceneID: sceneID,
		TrapID:  trapID,
	}

	result, err := e.TriggerTrap(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: result.TrapTriggered,
		Data:    result,
		Message: result.Message,
	}, nil
}
