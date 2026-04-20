package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 魔法物品工具 ==========

// UseMagicItemTool 使用魔法物品
type UseMagicItemTool struct {
	EngineTool
}

func NewUseMagicItemTool(e *engine.Engine) *UseMagicItemTool {
	return &UseMagicItemTool{
		EngineTool: *NewEngineTool(
			"use_magic_item",
			"使用魔法物品（消耗品或充能物品）",
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
					"item_id": map[string]any{
						"type":        "string",
						"description": "物品ID",
					},
					"target_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "目标ID列表（可选）",
					},
				},
				"required": []string{"game_id", "actor_id", "item_id"},
			},
			e,
			false,
		),
	}
}

func (t *UseMagicItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	itemIDStr, err := RequireString(params, "item_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)
	itemID := model.ID(itemIDStr)

	var targetIDs []model.ID
	targetStrs := OptionalStringArray(params, "target_ids")
	if len(targetStrs) > 0 {
		targetIDs = make([]model.ID, len(targetStrs))
		for i, tid := range targetStrs {
			targetIDs[i] = model.ID(tid)
		}
	}

	req := engine.UseMagicItemRequest{
		GameID:    gameID,
		ActorID:   actorID,
		ItemID:    itemID,
		TargetIDs: targetIDs,
	}

	result, err := e.UseMagicItem(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	msg := fmt.Sprintf("使用 %s", result.ItemName)
	if result.Consumed {
		msg += "（已消耗）"
	}
	if len(result.Messages) > 0 {
		msg += "：" + result.Messages[0]
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: msg,
	}, nil
}

// UnattuneItemTool 解除调谐
type UnattuneItemTool struct {
	EngineTool
}

func NewUnattuneItemTool(e *engine.Engine) *UnattuneItemTool {
	return &UnattuneItemTool{
		EngineTool: *NewEngineTool(
			"unattune_item",
			"解除魔法物品的调谐",
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
					"item_id": map[string]any{
						"type":        "string",
						"description": "物品ID",
					},
				},
				"required": []string{"game_id", "actor_id", "item_id"},
			},
			e,
			false,
		),
	}
}

func (t *UnattuneItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	itemIDStr, err := RequireString(params, "item_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)
	itemID := model.ID(itemIDStr)

	req := engine.UnattuneItemRequest{
		GameID:  gameID,
		ActorID: actorID,
		ItemID:  itemID,
	}

	result, err := e.UnattuneItem(ctx, req)
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

// RechargeMagicItemsTool 充能魔法物品
type RechargeMagicItemsTool struct {
	EngineTool
}

func NewRechargeMagicItemsTool(e *engine.Engine) *RechargeMagicItemsTool {
	return &RechargeMagicItemsTool{
		EngineTool: *NewEngineTool(
			"recharge_magic_items",
			"在黎明时恢复角色所有魔法物品的充能",
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

func (t *RechargeMagicItemsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)

	req := engine.RechargeMagicItemsRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	result, err := e.RechargeMagicItems(ctx, req)
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

// GetMagicItemBonusTool 获取魔法物品加值
type GetMagicItemBonusTool struct {
	EngineTool
}

func NewGetMagicItemBonusTool(e *engine.Engine) *GetMagicItemBonusTool {
	return &GetMagicItemBonusTool{
		EngineTool: *NewEngineTool(
			"get_magic_item_bonus",
			"获取魔法物品的加值信息",
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
					"item_id": map[string]any{
						"type":        "string",
						"description": "物品ID",
					},
				},
				"required": []string{"game_id", "actor_id", "item_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetMagicItemBonusTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	itemIDStr, err := RequireString(params, "item_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)
	itemID := model.ID(itemIDStr)

	req := engine.GetMagicItemBonusRequest{
		GameID:  gameID,
		ActorID: actorID,
		ItemID:  itemID,
	}

	result, err := e.GetMagicItemBonus(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	msg := fmt.Sprintf("%s：魔法加值 +%d", result.ItemName, result.MagicBonus)
	if result.Attuned {
		msg += "（已调谐）"
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: msg,
	}, nil
}
