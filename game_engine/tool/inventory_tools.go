package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 库存管理工具 ==========

// AddItemTool 添加物品到角色库存
type AddItemTool struct {
	EngineTool
}

func NewAddItemTool(e *engine.Engine) *AddItemTool {
	return &AddItemTool{
		EngineTool: *NewEngineTool(
			"add_item",
			"添加物品到角色库存",
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
					"item": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name":        map[string]any{"type": "string", "description": "物品名称"},
							"description": map[string]any{"type": "string", "description": "物品描述"},
							"type":        map[string]any{"type": "string", "description": "物品类型 (weapon, armor, potion, scroll, ring, wand, rod, staff, wondrous_item, ammunition, gear, tool)"},
							"rarity":      map[string]any{"type": "string", "description": "稀有度 (common, uncommon, rare, very_rare, legendary, artifact)"},
							"weight":      map[string]any{"type": "number", "description": "重量（磅）"},
							"quantity":    map[string]any{"type": "integer", "description": "数量"},
							"value":       map[string]any{"type": "integer", "description": "价值（铜币）"},
							"consumable":  map[string]any{"type": "boolean", "description": "是否为消耗品"},
							"effect":      map[string]any{"type": "string", "description": "使用效果"},
							"charges":     map[string]any{"type": "integer", "description": "充能次数"},
							"max_charges": map[string]any{"type": "integer", "description": "最大充能"},
							"recharge":    map[string]any{"type": "string", "description": "充能恢复条件"},
						},
						"required": []string{"name", "type"},
					},
				},
				"required": []string{"game_id", "actor_id", "item"},
			},
			e,
			false,
		),
	}
}

func (t *AddItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	itemData, err := RequireMap(params, "item")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)

	name, err := RequireString(itemData, "name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	item := &engine.ItemInput{
		Name: name,
	}

	item.Description = OptionalString(itemData, "description", "")
	item.Type = model.ItemType(OptionalString(itemData, "type", ""))
	item.Rarity = model.Rarity(OptionalString(itemData, "rarity", ""))
	item.Weight = OptionalFloat(itemData, "weight", 0)
	item.Quantity = OptionalInt(itemData, "quantity", 1)
	item.Value = OptionalInt(itemData, "value", 0)
	item.Consumable = OptionalBool(itemData, "consumable", false)
	item.Effect = OptionalString(itemData, "effect", "")
	item.Charges = OptionalInt(itemData, "charges", 0)
	item.MaxCharges = OptionalInt(itemData, "max_charges", 0)
	item.Recharge = OptionalString(itemData, "recharge", "")

	req := engine.AddItemRequest{
		GameID:  gameID,
		ActorID: actorID,
		Item:    item,
	}

	result, err := e.AddItem(ctx, req)
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

// RemoveItemTool 从角色库存移除物品
type RemoveItemTool struct {
	EngineTool
}

func NewRemoveItemTool(e *engine.Engine) *RemoveItemTool {
	return &RemoveItemTool{
		EngineTool: *NewEngineTool(
			"remove_item",
			"从角色库存移除物品",
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
					"quantity": map[string]any{
						"type":        "integer",
						"description": "移除数量",
					},
				},
				"required": []string{"game_id", "actor_id", "item_id"},
			},
			e,
			false,
		),
	}
}

func (t *RemoveItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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
	quantity := OptionalInt(params, "quantity", 1)

	req := engine.RemoveItemRequest{
		GameID:   gameID,
		ActorID:  actorID,
		ItemID:   itemID,
		Quantity: quantity,
	}

	result, err := e.RemoveItem(ctx, req)
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

// GetInventoryTool 获取角色库存
type GetInventoryTool struct {
	EngineTool
}

func NewGetInventoryTool(e *engine.Engine) *GetInventoryTool {
	return &GetInventoryTool{
		EngineTool: *NewEngineTool(
			"get_inventory",
			"获取角色背包中的所有物品列表，包括物品名称、数量、重量、类型等，以及总重量。Use when: 玩家询问'我包里有什么'；需要确认角色是否拥有某件物品；检查负重情况。Do NOT use when: 只需要查看当前装备的物品（用 get_equipment）；需要添加/移除物品（用 add_item/remove_item，这是 write 操作）。",
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

func (t *GetInventoryTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.GetInventoryRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	result, err := e.GetInventory(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("库存共 %d 件物品，总重量 %.1f 磅", len(result.Items), result.TotalWeight),
	}, nil
}

// EquipItemTool 装备物品
type EquipItemTool struct {
	EngineTool
}

func NewEquipItemTool(e *engine.Engine) *EquipItemTool {
	return &EquipItemTool{
		EngineTool: *NewEngineTool(
			"equip_item",
			"装备物品到指定槽位",
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
					"slot": map[string]any{
						"type":        "string",
						"description": "装备槽位 (main_hand, off_hand, chest, finger1, finger2, neck, head, back, hands, feet, waist)",
					},
				},
				"required": []string{"game_id", "actor_id", "item_id", "slot"},
			},
			e,
			false,
		),
	}
}

func (t *EquipItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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
	slotStr, err := RequireString(params, "slot")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)
	itemID := model.ID(itemIDStr)
	slot := model.EquipmentSlot(slotStr)

	req := engine.EquipItemRequest{
		GameID:  gameID,
		ActorID: actorID,
		ItemID:  itemID,
		Slot:    slot,
	}

	result, err := e.EquipItem(ctx, req)
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

// UnequipItemTool 卸下装备
type UnequipItemTool struct {
	EngineTool
}

func NewUnequipItemTool(e *engine.Engine) *UnequipItemTool {
	return &UnequipItemTool{
		EngineTool: *NewEngineTool(
			"unequip_item",
			"卸下指定槽位的装备",
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
					"slot": map[string]any{
						"type":        "string",
						"description": "装备槽位",
					},
				},
				"required": []string{"game_id", "actor_id", "slot"},
			},
			e,
			false,
		),
	}
}

func (t *UnequipItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	slotStr, err := RequireString(params, "slot")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)
	slot := model.EquipmentSlot(slotStr)

	req := engine.UnequipItemRequest{
		GameID:  gameID,
		ActorID: actorID,
		Slot:    slot,
	}

	result, err := e.UnequipItem(ctx, req)
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

// GetEquipmentTool 获取装备信息
type GetEquipmentTool struct {
	EngineTool
}

func NewGetEquipmentTool(e *engine.Engine) *GetEquipmentTool {
	return &GetEquipmentTool{
		EngineTool: *NewEngineTool(
			"get_equipment",
			"获取角色当前装备信息",
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

func (t *GetEquipmentTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.GetEquipmentRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	result, err := e.GetEquipment(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("当前装备 %d 个槽位，AC加值 %d", len(result.EquippedSlots), result.TotalACBonus),
	}, nil
}

// TransferItemTool 转移物品
type TransferItemTool struct {
	EngineTool
}

func NewTransferItemTool(e *engine.Engine) *TransferItemTool {
	return &TransferItemTool{
		EngineTool: *NewEngineTool(
			"transfer_item",
			"将物品从一个角色转移给另一个角色",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"from_actor_id": map[string]any{
						"type":        "string",
						"description": "源角色ID",
					},
					"to_actor_id": map[string]any{
						"type":        "string",
						"description": "目标角色ID",
					},
					"item_id": map[string]any{
						"type":        "string",
						"description": "物品ID",
					},
					"quantity": map[string]any{
						"type":        "integer",
						"description": "转移数量",
					},
				},
				"required": []string{"game_id", "from_actor_id", "to_actor_id", "item_id"},
			},
			e,
			false,
		),
	}
}

func (t *TransferItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	fromActorIDStr, err := RequireString(params, "from_actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	toActorIDStr, err := RequireString(params, "to_actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	itemIDStr, err := RequireString(params, "item_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	fromActorID := model.ID(fromActorIDStr)
	toActorID := model.ID(toActorIDStr)
	itemID := model.ID(itemIDStr)
	quantity := OptionalInt(params, "quantity", 1)

	req := engine.TransferItemRequest{
		GameID:      gameID,
		FromActorID: fromActorID,
		ToActorID:   toActorID,
		ItemID:      itemID,
		Quantity:    quantity,
	}

	result, err := e.TransferItem(ctx, req)
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

// AttuneItemTool 调谐魔法物品
type AttuneItemTool struct {
	EngineTool
}

func NewAttuneItemTool(e *engine.Engine) *AttuneItemTool {
	return &AttuneItemTool{
		EngineTool: *NewEngineTool(
			"attune_item",
			"调谐或解除调谐魔法物品",
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

func (t *AttuneItemTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.AttuneItemRequest{
		GameID:  gameID,
		ActorID: actorID,
		ItemID:  itemID,
	}

	result, err := e.AttuneItem(ctx, req)
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

// AddCurrencyTool 添加货币
type AddCurrencyTool struct {
	EngineTool
}

func NewAddCurrencyTool(e *engine.Engine) *AddCurrencyTool {
	return &AddCurrencyTool{
		EngineTool: *NewEngineTool(
			"add_currency",
			"添加货币到角色库存",
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
					"platinum": map[string]any{
						"type":        "integer",
						"description": "铂金币数量",
					},
					"gold": map[string]any{
						"type":        "integer",
						"description": "金币数量",
					},
					"electrum": map[string]any{
						"type":        "integer",
						"description": "银金币数量",
					},
					"silver": map[string]any{
						"type":        "integer",
						"description": "银币数量",
					},
					"copper": map[string]any{
						"type":        "integer",
						"description": "铜币数量",
					},
				},
				"required": []string{"game_id", "actor_id"},
			},
			e,
			false,
		),
	}
}

func (t *AddCurrencyTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	currency := model.Currency{
		Platinum: OptionalInt(params, "platinum", 0),
		Gold:     OptionalInt(params, "gold", 0),
		Electrum: OptionalInt(params, "electrum", 0),
		Silver:   OptionalInt(params, "silver", 0),
		Copper:   OptionalInt(params, "copper", 0),
	}

	req := engine.AddCurrencyRequest{
		GameID:   gameID,
		ActorID:  actorID,
		Currency: currency,
	}

	result, err := e.AddCurrency(ctx, req)
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
