package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 制作工具 ==========

// StartCraftingTool 开始制作物品
type StartCraftingTool struct {
	EngineTool
}

func NewStartCraftingTool(e *engine.Engine) *StartCraftingTool {
	return &StartCraftingTool{
		EngineTool: *NewEngineTool(
			"start_crafting",
			"开始制作物品（验证等级、工具熟练、金币后创建制作进度）",
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
					"recipe_id": map[string]any{
						"type":        "string",
						"description": "配方ID",
					},
				},
				"required": []string{"game_id", "actor_id", "recipe_id"},
			},
			e,
			false,
		),
	}
}

func (t *StartCraftingTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	recipeID, err := RequireString(params, "recipe_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.StartCraftingRequest{
		GameID:   gameID,
		ActorID:  actorID,
		RecipeID: recipeID,
	}

	result, err := e.StartCrafting(ctx, req)
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

// AdvanceCraftingTool 推进制作进度
type AdvanceCraftingTool struct {
	EngineTool
}

func NewAdvanceCraftingTool(e *engine.Engine) *AdvanceCraftingTool {
	return &AdvanceCraftingTool{
		EngineTool: *NewEngineTool(
			"advance_crafting",
			"推进制作进度（按天数计算）",
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
					"days": map[string]any{
						"type":        "integer",
						"description": "制作天数",
					},
				},
				"required": []string{"game_id", "actor_id", "days"},
			},
			e,
			false,
		),
	}
}

func (t *AdvanceCraftingTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	days, err := RequireInt(params, "days")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.AdvanceCraftingRequest{
		GameID:  gameID,
		ActorID: actorID,
		Days:    days,
	}

	result, err := e.AdvanceCrafting(ctx, req)
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

// CompleteCraftingTool 完成制作
type CompleteCraftingTool struct {
	EngineTool
}

func NewCompleteCraftingTool(e *engine.Engine) *CompleteCraftingTool {
	return &CompleteCraftingTool{
		EngineTool: *NewEngineTool(
			"complete_crafting",
			"完成制作并获取物品",
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
					"recipe_id": map[string]any{
						"type":        "string",
						"description": "配方ID",
					},
				},
				"required": []string{"game_id", "actor_id", "recipe_id"},
			},
			e,
			false,
		),
	}
}

func (t *CompleteCraftingTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	recipeID, err := RequireString(params, "recipe_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.CompleteCraftingRequest{
		GameID:   gameID,
		ActorID:  actorID,
		RecipeID: recipeID,
	}

	result, err := e.CompleteCrafting(ctx, req)
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

// GetCraftingRecipesTool 获取制作配方列表
type GetCraftingRecipesTool struct {
	EngineTool
}

func NewGetCraftingRecipesTool(e *engine.Engine) *GetCraftingRecipesTool {
	return &GetCraftingRecipesTool{
		EngineTool: *NewEngineTool(
			"get_crafting_recipes",
			"获取所有可用的制作配方",
			map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			e,
			true,
		),
	}
}

func (t *GetCraftingRecipesTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	result, err := e.GetCraftingRecipes(ctx)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个配方", len(result)),
	}, nil
}
