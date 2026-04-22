package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// =============================================================================
// 18. manage_item - 物品管理统一入口
// =============================================================================

type ManageItemTool struct {
	EngineTool
}

func NewManageItemTool(e *engine.Engine) *ManageItemTool {
	return &ManageItemTool{
		EngineTool: *NewEngineTool(
			"manage_item",
			"物品增删和转移的统一操作入口。支持添加物品、移除物品、转移物品、添加货币",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":  map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_id": map[string]any{"type": "string", "description": "角色ID"},
					"operation": map[string]any{
						"type": "string", "enum": []string{"add", "remove", "transfer", "add_currency"},
						"description": "操作类型",
					},
					"item": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name":        map[string]any{"type": "string", "description": "物品名称"},
							"type":        map[string]any{"type": "string", "description": "物品类型"},
							"description": map[string]any{"type": "string"},
							"rarity":      map[string]any{"type": "string"},
							"weight":      map[string]any{"type": "number"},
							"quantity":    map[string]any{"type": "integer"},
							"value":       map[string]any{"type": "integer"},
							"consumable":  map[string]any{"type": "boolean"},
							"effect":      map[string]any{"type": "string"},
							"charges":     map[string]any{"type": "integer"},
							"max_charges": map[string]any{"type": "integer"},
							"recharge":    map[string]any{"type": "string"},
						},
						"required":    []string{"name"},
						"description": "物品信息（add时必需）",
					},
					"item_id":         map[string]any{"type": "string", "description": "物品ID（remove/transfer时必需）"},
					"quantity":        map[string]any{"type": "integer", "description": "数量（可选，默认1）"},
					"target_actor_id": map[string]any{"type": "string", "description": "目标角色ID（transfer时必需）"},
					"currency": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"platinum": map[string]any{"type": "integer"},
							"gold":     map[string]any{"type": "integer"},
							"electrum": map[string]any{"type": "integer"},
							"silver":   map[string]any{"type": "integer"},
							"copper":   map[string]any{"type": "integer"},
						},
						"description": "货币（add_currency时必需）",
					},
				},
				"required": []string{"game_id", "actor_id", "operation"},
			},
			e,
			false,
		),
	}
}

func (t *ManageItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	op, err := RequireString(params, "operation")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	switch op {
	case "add":
		itemData, ierr := RequireMap(params, "item")
		if ierr != nil {
			return &ToolResult{Success: false, Error: "add需要item参数"}, nil
		}
		itemName, nerr := RequireString(itemData, "name")
		if nerr != nil {
			return &ToolResult{Success: false, Error: "item需要name字段"}, nil
		}
		item := &engine.ItemInput{
			Name:        itemName,
			Description: OptionalString(itemData, "description", ""),
			Type:        model.ItemType(OptionalString(itemData, "type", "")),
			Rarity:      model.Rarity(OptionalString(itemData, "rarity", "")),
			Weight:      OptionalFloat(itemData, "weight", 0),
			Quantity:    OptionalInt(itemData, "quantity", 1),
			Value:       OptionalInt(itemData, "value", 0),
			Consumable:  OptionalBool(itemData, "consumable", false),
			Effect:      OptionalString(itemData, "effect", ""),
			Charges:     OptionalInt(itemData, "charges", 0),
			MaxCharges:  OptionalInt(itemData, "max_charges", 0),
			Recharge:    OptionalString(itemData, "recharge", ""),
		}
		result, aerr := e.AddItem(ctx, engine.AddItemRequest{GameID: gameID, ActorID: actorID, Item: item})
		if aerr != nil {
			return &ToolResult{Success: false, Error: aerr.Error()}, nil
		}
		return &ToolResult{Success: result.Success, Data: result, Message: result.Message}, nil

	case "remove":
		itemIDStr, ierr := RequireString(params, "item_id")
		if ierr != nil {
			return &ToolResult{Success: false, Error: "remove需要item_id"}, nil
		}
		result, rerr := e.RemoveItem(ctx, engine.RemoveItemRequest{
			GameID: gameID, ActorID: actorID, ItemID: model.ID(itemIDStr),
			Quantity: OptionalInt(params, "quantity", 1),
		})
		if rerr != nil {
			return &ToolResult{Success: false, Error: rerr.Error()}, nil
		}
		return &ToolResult{Success: result.Success, Data: result, Message: result.Message}, nil

	case "transfer":
		itemIDStr, ierr := RequireString(params, "item_id")
		if ierr != nil {
			return &ToolResult{Success: false, Error: "transfer需要item_id"}, nil
		}
		targetIDStr, terr := RequireString(params, "target_actor_id")
		if terr != nil {
			return &ToolResult{Success: false, Error: "transfer需要target_actor_id"}, nil
		}
		result, trerr := e.TransferItem(ctx, engine.TransferItemRequest{
			GameID: gameID, FromActorID: actorID, ToActorID: model.ID(targetIDStr),
			ItemID: model.ID(itemIDStr), Quantity: OptionalInt(params, "quantity", 1),
		})
		if trerr != nil {
			return &ToolResult{Success: false, Error: trerr.Error()}, nil
		}
		return &ToolResult{Success: result.Success, Data: result, Message: result.Message}, nil

	case "add_currency":
		currData, cerr := RequireMap(params, "currency")
		if cerr != nil {
			return &ToolResult{Success: false, Error: "add_currency需要currency参数"}, nil
		}
		currency := model.Currency{
			Platinum: OptionalInt(currData, "platinum", 0),
			Gold:     OptionalInt(currData, "gold", 0),
			Electrum: OptionalInt(currData, "electrum", 0),
			Silver:   OptionalInt(currData, "silver", 0),
			Copper:   OptionalInt(currData, "copper", 0),
		}
		result, aderr := e.AddCurrency(ctx, engine.AddCurrencyRequest{
			GameID: gameID, ActorID: actorID, Currency: currency,
		})
		if aderr != nil {
			return &ToolResult{Success: false, Error: aderr.Error()}, nil
		}
		return &ToolResult{Success: result.Success, Data: result, Message: result.Message}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效操作: %s", op)}, nil
	}
}

// =============================================================================
// 19. equip_item - 装备管理
// =============================================================================

type EquipItemCompositeTool struct {
	EngineTool
}

func NewEquipItemCompositeTool(e *engine.Engine) *EquipItemCompositeTool {
	return &EquipItemCompositeTool{
		EngineTool: *NewEngineTool(
			"equip_item",
			"装备/卸下/调谐物品。支持一键装备并调谐",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":  map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_id": map[string]any{"type": "string", "description": "角色ID"},
					"item_id":  map[string]any{"type": "string", "description": "物品ID"},
					"operation": map[string]any{
						"type": "string", "enum": []string{"equip", "unequip", "attune", "unattune", "equip_and_attune"},
						"description": "操作类型",
					},
					"slot": map[string]any{"type": "string", "description": "装备槽位（equip/unequip时必需）"},
				},
				"required": []string{"game_id", "actor_id", "operation"},
			},
			e,
			false,
		),
	}
}

func (t *EquipItemCompositeTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	op, err := RequireString(params, "operation")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	switch op {
	case "equip":
		itemIDStr, ierr := RequireString(params, "item_id")
		if ierr != nil {
			return &ToolResult{Success: false, Error: "equip需要item_id"}, nil
		}
		slotStr, serr := RequireString(params, "slot")
		if serr != nil {
			return &ToolResult{Success: false, Error: "equip需要slot"}, nil
		}
		result, eerr := e.EquipItem(ctx, engine.EquipItemRequest{
			GameID: gameID, ActorID: actorID, ItemID: model.ID(itemIDStr), Slot: model.EquipmentSlot(slotStr),
		})
		if eerr != nil {
			return &ToolResult{Success: false, Error: eerr.Error()}, nil
		}
		return &ToolResult{Success: result.Success, Data: result, Message: result.Message}, nil

	case "unequip":
		slotStr, serr := RequireString(params, "slot")
		if serr != nil {
			return &ToolResult{Success: false, Error: "unequip需要slot"}, nil
		}
		result, uerr := e.UnequipItem(ctx, engine.UnequipItemRequest{
			GameID: gameID, ActorID: actorID, Slot: model.EquipmentSlot(slotStr),
		})
		if uerr != nil {
			return &ToolResult{Success: false, Error: uerr.Error()}, nil
		}
		return &ToolResult{Success: result.Success, Data: result, Message: result.Message}, nil

	case "attune":
		itemIDStr, ierr := RequireString(params, "item_id")
		if ierr != nil {
			return &ToolResult{Success: false, Error: "attune需要item_id"}, nil
		}
		result, aerr := e.AttuneItem(ctx, engine.AttuneItemRequest{
			GameID: gameID, ActorID: actorID, ItemID: model.ID(itemIDStr),
		})
		if aerr != nil {
			return &ToolResult{Success: false, Error: aerr.Error()}, nil
		}
		return &ToolResult{Success: result.Success, Data: result, Message: result.Message}, nil

	case "unattune":
		itemIDStr, ierr := RequireString(params, "item_id")
		if ierr != nil {
			return &ToolResult{Success: false, Error: "unattune需要item_id"}, nil
		}
		result, uerr := e.UnattuneItem(ctx, engine.UnattuneItemRequest{
			GameID: gameID, ActorID: actorID, ItemID: model.ID(itemIDStr),
		})
		if uerr != nil {
			return &ToolResult{Success: false, Error: uerr.Error()}, nil
		}
		return &ToolResult{Success: result.Success, Data: result, Message: result.Message}, nil

	case "equip_and_attune":
		itemIDStr, ierr := RequireString(params, "item_id")
		if ierr != nil {
			return &ToolResult{Success: false, Error: "equip_and_attune需要item_id"}, nil
		}
		slotStr, serr := RequireString(params, "slot")
		if serr != nil {
			return &ToolResult{Success: false, Error: "equip_and_attune需要slot"}, nil
		}
		itemID := model.ID(itemIDStr)
		equipResult, eerr := e.EquipItem(ctx, engine.EquipItemRequest{
			GameID: gameID, ActorID: actorID, ItemID: itemID, Slot: model.EquipmentSlot(slotStr),
		})
		if eerr != nil {
			return &ToolResult{Success: false, Error: eerr.Error()}, nil
		}
		attuneResult, aerr := e.AttuneItem(ctx, engine.AttuneItemRequest{
			GameID: gameID, ActorID: actorID, ItemID: itemID,
		})
		if aerr != nil {
			return &ToolResult{
				Success: true,
				Data:    map[string]any{"equip": equipResult, "attune_error": aerr.Error()},
				Message: equipResult.Message + "（调谐失败: " + aerr.Error() + "）",
			}, nil
		}
		return &ToolResult{
			Success: true,
			Data:    map[string]any{"equip": equipResult, "attune": attuneResult},
			Message: "装备并调谐成功",
		}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效操作: %s", op)}, nil
	}
}

// =============================================================================
// 20. use_item - 使用物品
// =============================================================================

type UseItemTool struct {
	EngineTool
}

func NewUseItemTool(e *engine.Engine) *UseItemTool {
	return &UseItemTool{
		EngineTool: *NewEngineTool(
			"use_item",
			"使用魔法物品或消耗品",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":    map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_id":   map[string]any{"type": "string", "description": "使用者角色ID"},
					"item_id":    map[string]any{"type": "string", "description": "物品ID"},
					"target_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "目标ID列表（可选）"},
				},
				"required": []string{"game_id", "actor_id", "item_id"},
			},
			e,
			false,
		),
	}
}

func (t *UseItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	result, uerr := e.UseMagicItem(ctx, engine.UseMagicItemRequest{
		GameID: gameID, ActorID: actorID, ItemID: itemID, TargetIDs: targetIDs,
	})
	if uerr != nil {
		return &ToolResult{Success: false, Error: uerr.Error()}, nil
	}

	msg := fmt.Sprintf("使用 %s", result.ItemName)
	if result.Consumed {
		msg += "（已消耗）"
	}
	if len(result.Messages) > 0 {
		msg += "：" + result.Messages[0]
	}

	return &ToolResult{Success: true, Data: result, Message: msg}, nil
}

// =============================================================================
// 21. query_inventory - 库存查询
// =============================================================================

type QueryInventoryTool struct {
	EngineTool
}

func NewQueryInventoryTool(e *engine.Engine) *QueryInventoryTool {
	return &QueryInventoryTool{
		EngineTool: *NewEngineTool(
			"query_inventory",
			"查询角色库存、装备和魔法物品加值信息",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":                map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_id":               map[string]any{"type": "string", "description": "角色ID"},
					"include_equipment":      map[string]any{"type": "boolean", "description": "是否包含装备信息（默认true）"},
					"include_magic_bonuses": map[string]any{"type": "boolean", "description": "是否包含魔法物品加值（默认false）"},
				},
				"required": []string{"game_id", "actor_id"},
			},
			e,
			true, // 只读
		),
	}
}

func (t *QueryInventoryTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	invResult, ierr := e.GetInventory(ctx, engine.GetInventoryRequest{GameID: gameID, ActorID: actorID})
	if ierr != nil {
		return &ToolResult{Success: false, Error: ierr.Error()}, nil
	}
	data["inventory"] = invResult
	msg := fmt.Sprintf("库存 %d 件物品", len(invResult.Items))

	if OptionalBool(params, "include_equipment", true) {
		eqResult, eerr := e.GetEquipment(ctx, engine.GetEquipmentRequest{GameID: gameID, ActorID: actorID})
		if eerr == nil {
			data["equipment"] = eqResult
			msg += fmt.Sprintf("，装备 %d 个槽位", len(eqResult.EquippedSlots))
		}
	}

	return &ToolResult{
		Success: true,
		Data:    data,
		Message: msg,
	}, nil
}
