package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 场景管理工具 ==========

// CreateSceneTool 创建场景
type CreateSceneTool struct {
	EngineTool
}

func NewCreateSceneTool(e *engine.Engine) *CreateSceneTool {
	return &CreateSceneTool{
		EngineTool: *NewEngineTool(
			"create_scene",
			"创建新的游戏场景",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "场景名称",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "场景描述",
					},
					"scene_type": map[string]any{
						"type":        "string",
						"description": "场景类型 (indoor, outdoor, wilderness, dungeon, city, tavern, shop, temple, other)",
					},
				},
				"required": []string{"game_id", "name", "description", "scene_type"},
			},
			e,
			false,
		),
	}
}

func (t *CreateSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	name, err := RequireString(params, "name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	description, err := RequireString(params, "description")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	sceneTypeStr, err := RequireString(params, "scene_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	sceneType := model.SceneType(sceneTypeStr)

	req := engine.CreateSceneRequest{
		GameID:      gameID,
		Name:        name,
		Description: description,
		SceneType:   sceneType,
	}

	result, err := e.CreateScene(ctx, req)
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

// GetSceneTool 获取场景信息
type GetSceneTool struct {
	EngineTool
}

func NewGetSceneTool(e *engine.Engine) *GetSceneTool {
	return &GetSceneTool{
		EngineTool: *NewEngineTool(
			"get_scene",
			"获取指定场景的详细信息，包括名称、描述、连接关系等。Use when: 需要了解某个场景的具体描述和配置；玩家询问特定地点的信息。Do NOT use when: 需要获取当前所在场景（用 get_current_scene）；需要列出所有场景（用 list_scenes）；需要查看场景中的角色或物品（用 get_scene_actors/get_scene_items）。",
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
				},
				"required": []string{"game_id", "scene_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.GetSceneRequest{
		GameID:  gameID,
		SceneID: sceneID,
	}

	result, err := e.GetScene(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("场景: %s (%s), 连接数: %d", result.Name, result.Type, len(result.Connections)),
	}, nil
}

// UpdateSceneTool 更新场景信息
type UpdateSceneTool struct {
	EngineTool
}

func NewUpdateSceneTool(e *engine.Engine) *UpdateSceneTool {
	return &UpdateSceneTool{
		EngineTool: *NewEngineTool(
			"update_scene",
			"更新场景的属性信息（名称、描述、光照、天气、地形等）",
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
					"name": map[string]any{
						"type":        "string",
						"description": "场景名称（可选）",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "场景描述（可选）",
					},
					"details": map[string]any{
						"type":        "string",
						"description": "场景细节（可选）",
					},
					"is_dark": map[string]any{
						"type":        "boolean",
						"description": "是否为黑暗环境（可选）",
					},
					"light_level": map[string]any{
						"type":        "string",
						"description": "光照等级 (bright, dim, darkness)（可选）",
					},
					"weather": map[string]any{
						"type":        "string",
						"description": "天气状况（可选）",
					},
					"terrain": map[string]any{
						"type":        "string",
						"description": "地形类型（可选）",
					},
				},
				"required": []string{"game_id", "scene_id"},
			},
			e,
			false,
		),
	}
}

func (t *UpdateSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	updates := engine.SceneUpdate{}
	if name := OptionalString(params, "name", ""); name != "" {
		updates.Name = name
	}
	if desc := OptionalString(params, "description", ""); desc != "" {
		updates.Description = desc
	}
	if details := OptionalString(params, "details", ""); details != "" {
		updates.Details = details
	}
	if _, ok := params["is_dark"]; ok {
		isDark := OptionalBool(params, "is_dark", false)
		updates.IsDark = isDark
		updates.IsDarkSet = true
	}
	if lightLevel := OptionalString(params, "light_level", ""); lightLevel != "" {
		updates.LightLevel = lightLevel
	}
	if weather := OptionalString(params, "weather", ""); weather != "" {
		updates.Weather = weather
	}
	if terrain := OptionalString(params, "terrain", ""); terrain != "" {
		updates.Terrain = terrain
	}

	req := engine.UpdateSceneRequest{
		GameID:  gameID,
		SceneID: sceneID,
		Updates: updates,
	}

	err = e.UpdateScene(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("场景 %s 已更新", sceneID),
	}, nil
}

// DeleteSceneTool 删除场景
type DeleteSceneTool struct {
	EngineTool
}

func NewDeleteSceneTool(e *engine.Engine) *DeleteSceneTool {
	return &DeleteSceneTool{
		EngineTool: *NewEngineTool(
			"delete_scene",
			"删除指定场景（场景中有角色时无法删除）",
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
				},
				"required": []string{"game_id", "scene_id"},
			},
			e,
			false,
		),
	}
}

func (t *DeleteSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.DeleteSceneRequest{
		GameID:  gameID,
		SceneID: sceneID,
	}

	err = e.DeleteScene(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("场景 %s 已删除", sceneID),
	}, nil
}

// ListScenesTool 列出所有场景
type ListScenesTool struct {
	EngineTool
}

func NewListScenesTool(e *engine.Engine) *ListScenesTool {
	return &ListScenesTool{
		EngineTool: *NewEngineTool(
			"list_scenes",
			"列出游戏中的所有场景",
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

func (t *ListScenesTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	req := engine.ListScenesRequest{
		GameID: gameID,
	}

	result, err := e.ListScenes(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个场景", len(result.Scenes)),
	}, nil
}

// SetCurrentSceneTool 设置当前场景
type SetCurrentSceneTool struct {
	EngineTool
}

func NewSetCurrentSceneTool(e *engine.Engine) *SetCurrentSceneTool {
	return &SetCurrentSceneTool{
		EngineTool: *NewEngineTool(
			"set_current_scene",
			"设置当前活跃场景",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"scene_id": map[string]any{
						"type":        "string",
						"description": "要设置为当前场景的ID",
					},
				},
				"required": []string{"game_id", "scene_id"},
			},
			e,
			false,
		),
	}
}

func (t *SetCurrentSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.SetCurrentSceneRequest{
		GameID:  gameID,
		SceneID: sceneID,
	}

	err = e.SetCurrentScene(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("已将场景 %s 设为当前场景", sceneID),
	}, nil
}

// GetCurrentSceneTool 获取当前场景
type GetCurrentSceneTool struct {
	EngineTool
}

func NewGetCurrentSceneTool(e *engine.Engine) *GetCurrentSceneTool {
	return &GetCurrentSceneTool{
		EngineTool: *NewEngineTool(
			"get_current_scene",
			"获取当前角色所在场景的详细信息，包括场景名称、描述、连接关系等。Use when: 需要描述玩家当前所在位置；需要确认当前场景的配置。Do NOT use when: 需要获取指定场景（用 get_scene）；需要查看场景中的角色或物品（用 get_scene_actors/get_scene_items）。",
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

func (t *GetCurrentSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	req := engine.GetCurrentSceneRequest{
		GameID: gameID,
	}

	result, err := e.GetCurrentScene(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("当前场景: %s (%s)", result.Name, result.Type),
	}, nil
}

// AddSceneConnectionTool 添加场景连接
type AddSceneConnectionTool struct {
	EngineTool
}

func NewAddSceneConnectionTool(e *engine.Engine) *AddSceneConnectionTool {
	return &AddSceneConnectionTool{
		EngineTool: *NewEngineTool(
			"add_scene_connection",
			"在两个场景之间创建连接通道",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"scene_id": map[string]any{
						"type":        "string",
						"description": "源场景ID",
					},
					"target_scene_id": map[string]any{
						"type":        "string",
						"description": "目标场景ID",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "连接描述（如：一条狭窄的走廊）",
					},
					"locked": map[string]any{
						"type":        "boolean",
						"description": "是否锁定（需要解锁才能通过）",
					},
					"dc": map[string]any{
						"type":        "integer",
						"description": "解锁难度等级（锁定时使用）",
					},
					"hidden": map[string]any{
						"type":        "boolean",
						"description": "是否为隐藏通道",
					},
				},
				"required": []string{"game_id", "scene_id", "target_scene_id", "description"},
			},
			e,
			false,
		),
	}
}

func (t *AddSceneConnectionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	targetSceneIDStr, err := RequireString(params, "target_scene_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	targetSceneID := model.ID(targetSceneIDStr)

	description, err := RequireString(params, "description")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	locked := OptionalBool(params, "locked", false)
	dc := OptionalInt(params, "dc", 0)
	hidden := OptionalBool(params, "hidden", false)

	req := engine.AddSceneConnectionRequest{
		GameID:        gameID,
		SceneID:       sceneID,
		TargetSceneID: targetSceneID,
		Description:   description,
		Locked:        locked,
		DC:            dc,
		Hidden:        hidden,
	}

	err = e.AddSceneConnection(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("已添加场景连接: %s -> %s", sceneID, targetSceneID),
	}, nil
}

// RemoveSceneConnectionTool 移除场景连接
type RemoveSceneConnectionTool struct {
	EngineTool
}

func NewRemoveSceneConnectionTool(e *engine.Engine) *RemoveSceneConnectionTool {
	return &RemoveSceneConnectionTool{
		EngineTool: *NewEngineTool(
			"remove_scene_connection",
			"移除两个场景之间的连接",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"scene_id": map[string]any{
						"type":        "string",
						"description": "源场景ID",
					},
					"target_scene_id": map[string]any{
						"type":        "string",
						"description": "目标场景ID",
					},
				},
				"required": []string{"game_id", "scene_id", "target_scene_id"},
			},
			e,
			false,
		),
	}
}

func (t *RemoveSceneConnectionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	targetSceneIDStr, err := RequireString(params, "target_scene_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	targetSceneID := model.ID(targetSceneIDStr)

	req := engine.RemoveSceneConnectionRequest{
		GameID:        gameID,
		SceneID:       sceneID,
		TargetSceneID: targetSceneID,
	}

	err = e.RemoveSceneConnection(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("已移除场景连接: %s -> %s", sceneID, targetSceneID),
	}, nil
}

// MoveActorToSceneTool 移动角色到场景
type MoveActorToSceneTool struct {
	EngineTool
}

func NewMoveActorToSceneTool(e *engine.Engine) *MoveActorToSceneTool {
	return &MoveActorToSceneTool{
		EngineTool: *NewEngineTool(
			"move_actor_to_scene",
			"将角色移动到指定场景",
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
						"description": "目标场景ID",
					},
				},
				"required": []string{"game_id", "actor_id", "scene_id"},
			},
			e,
			false,
		),
	}
}

func (t *MoveActorToSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.MoveActorToSceneRequest{
		GameID:  gameID,
		ActorID: actorID,
		SceneID: sceneID,
	}

	result, err := e.MoveActorToScene(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: result.SceneMoveResult.Success,
		Data:    result,
		Message: result.SceneMoveResult.Message,
	}, nil
}

// GetSceneActorsTool 获取场景中的角色
type GetSceneActorsTool struct {
	EngineTool
}

func NewGetSceneActorsTool(e *engine.Engine) *GetSceneActorsTool {
	return &GetSceneActorsTool{
		EngineTool: *NewEngineTool(
			"get_scene_actors",
			"获取指定场景中的所有角色",
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
				},
				"required": []string{"game_id", "scene_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetSceneActorsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.GetSceneActorsRequest{
		GameID:  gameID,
		SceneID: sceneID,
	}

	result, err := e.GetSceneActors(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("场景中有 %d 个角色", len(result.Actors)),
	}, nil
}

// AddItemToSceneTool 添加物品到场景
type AddItemToSceneTool struct {
	EngineTool
}

func NewAddItemToSceneTool(e *engine.Engine) *AddItemToSceneTool {
	return &AddItemToSceneTool{
		EngineTool: *NewEngineTool(
			"add_item_to_scene",
			"将物品放置到场景中",
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
					"item_id": map[string]any{
						"type":        "string",
						"description": "物品ID",
					},
				},
				"required": []string{"game_id", "scene_id", "item_id"},
			},
			e,
			false,
		),
	}
}

func (t *AddItemToSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	itemIDStr, err := RequireString(params, "item_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	itemID := model.ID(itemIDStr)

	req := engine.AddItemToSceneRequest{
		GameID:  gameID,
		SceneID: sceneID,
		ItemID:  itemID,
	}

	err = e.AddItemToScene(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("已将物品 %s 放置到场景 %s", itemID, sceneID),
	}, nil
}

// RemoveItemFromSceneTool 从场景移除物品
type RemoveItemFromSceneTool struct {
	EngineTool
}

func NewRemoveItemFromSceneTool(e *engine.Engine) *RemoveItemFromSceneTool {
	return &RemoveItemFromSceneTool{
		EngineTool: *NewEngineTool(
			"remove_item_from_scene",
			"从场景中移除物品",
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
					"item_id": map[string]any{
						"type":        "string",
						"description": "物品ID",
					},
				},
				"required": []string{"game_id", "scene_id", "item_id"},
			},
			e,
			false,
		),
	}
}

func (t *RemoveItemFromSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	itemIDStr, err := RequireString(params, "item_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	itemID := model.ID(itemIDStr)

	req := engine.RemoveItemFromSceneRequest{
		GameID:  gameID,
		SceneID: sceneID,
		ItemID:  itemID,
	}

	err = e.RemoveItemFromScene(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("已从场景 %s 移除物品 %s", sceneID, itemID),
	}, nil
}

// GetSceneItemsTool 获取场景物品
type GetSceneItemsTool struct {
	EngineTool
}

func NewGetSceneItemsTool(e *engine.Engine) *GetSceneItemsTool {
	return &GetSceneItemsTool{
		EngineTool: *NewEngineTool(
			"get_scene_items",
			"获取场景中的所有物品",
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
				},
				"required": []string{"game_id", "scene_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetSceneItemsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.GetSceneItemsRequest{
		GameID:  gameID,
		SceneID: sceneID,
	}

	result, err := e.GetSceneItems(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("场景中有 %d 件物品", len(result.Items)),
	}, nil
}

// ========== 复合工具 - 场景系统 ==========

// NewCreateConnectedSceneTool 复合创建连接场景工具：创建场景 + 自动添加与当前场景的连接
//
// 合并: create_scene + add_scene_connection
func NewCreateConnectedSceneTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"game_id": map[string]any{
				"type":        "string",
				"description": "游戏会话ID",
			},
			"current_scene_id": map[string]any{
				"type":        "string",
				"description": "当前场景ID（新场景将与这个场景建立连接）",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "新场景名称",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "新场景描述",
			},
			"scene_type": map[string]any{
				"type":        "string",
				"description": "场景类型 (indoor, outdoor, wilderness, dungeon, city, tavern, shop, temple, other)",
			},
			"connection_description": map[string]any{
				"type":        "string",
				"description": "连接描述（如：一条通往东边的门）",
			},
		},
		"required": []string{"game_id", "current_scene_id", "name", "description", "scene_type", "connection_description"},
	}

	desc := `Create a new scene and automatically create a connection from the current scene.

Use when: Discovering a new area that needs to be added connected to where the party currently is. One call creates the scene and connects it.

Do NOT use when: Just moving between existing connected scenes (no creation needed).

Parameters:
  - game_id: Game session ID
  - current_scene_id: ID of the current scene (the one the party is in)
  - name: Name of the new scene
  - description: Description of the new scene
  - scene_type: Type of scene
  - connection_description: Description of the connection from current to new

Returns: New scene created with connection established.`

	steps := []ToolStep{
		{
			ToolName: "create_scene",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return map[string]any{
					"game_id":         params["game_id"],
					"name":           params["name"],
					"description":    params["description"],
					"scene_type":     params["scene_type"],
				}
			},
		},
	}

	return NewCompositeTool(
		"create_connected_scene",
		desc,
		schema,
		registry,
		steps,
		false,
	)
}

// NewShowSceneDetailTool 复合显示场景详情：获取场景信息 + 获取角色 + 获取物品
//
// 合并: get_scene + get_scene_actors + get_scene_items
func NewShowSceneDetailTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
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
		},
		"required": []string{"game_id", "scene_id"},
	}

	desc := `Show complete scene detail including scene info, all actors, and all items in the scene.

Use when: Players ask for a detailed description of a scene, or you need complete information about what's present. One call gets everything.

Do NOT use when: You only need basic scene info (use get_scene), or just need actors (use get_scene_actors).

Parameters:
  - game_id: Game session ID
  - scene_id: Scene ID to query

Returns: Complete scene information with name, description, type, actors list, and items list.`

	steps := []ToolStep{
		{
			ToolName: "get_scene",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return params
			},
		},
		{
			ToolName: "get_scene_actors",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return params
			},
		},
		{
			ToolName: "get_scene_items",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return params
			},
		},
	}

	return NewCompositeTool(
		"show_scene_detail",
		desc,
		schema,
		registry,
		steps,
		true,
	)
}

// NewMoveToSceneTool 复合移动到场景：移动角色 + 获取新场景详情
//
// 合并: move_actor_to_scene + set_current_scene + show_scene_detail
func NewMoveToSceneTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"game_id": map[string]any{
				"type":        "string",
				"description": "游戏会话ID",
			},
			"actor_id": map[string]any{
				"type":        "string",
				"description": "要移动的角色ID（通常是玩家）",
			},
			"target_scene_id": map[string]any{
				"type":        "string",
				"description": "目标场景ID",
			},
		},
		"required": []string{"game_id", "actor_id", "target_scene_id"},
	}

	desc := `Move an actor to a target scene and return complete information about the new scene.

Use when: Player moves from one scene to another through a connection. One call completes the move and shows the new scene.

Do NOT use when: Just moving position within the same scene.

Parameters:
  - game_id: Game session ID
  - actor_id: Actor to move (usually the player character)
  - target_scene_id: Target scene ID to move into

Returns: Move complete with full scene detail of the destination.`

	steps := []ToolStep{
		{
			ToolName: "move_actor_to_scene",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return map[string]any{
					"game_id":   params["game_id"],
					"actor_id":  params["actor_id"],
					"scene_id":  params["target_scene_id"],
				}
			},
		},
		{
			ToolName: "set_current_scene",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return map[string]any{
					"game_id":  params["game_id"],
					"scene_id": params["target_scene_id"],
				}
			},
		},
		{
			ToolName: "show_scene_detail",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return map[string]any{
					"game_id":   params["game_id"],
					"scene_id":  params["target_scene_id"],
				}
			},
		},
	}

	return NewCompositeTool(
		"move_to_scene",
		desc,
		schema,
		registry,
		steps,
		false,
	)
}
