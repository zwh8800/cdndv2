package agent

import (
	"strings"

	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// InventoryAgent 库存、装备和制作管理Agent
type InventoryAgent struct {
	*BaseSubAgent
}

// NewInventoryAgent 创建库存管理Agent
func NewInventoryAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *InventoryAgent {
	return &InventoryAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameInventory,
			Description:  "库存管理Agent，负责物品管理、装备管理、魔法物品、货币管理、制作系统",
			TemplateFile: "inventory_system.md",
			DomainIntro:  "你是D&D 5e库存、装备和制作管理专家。",
			DomainRule:   "所有库存操作必须通过调用Tools完成，不得自行计算。",
			KeyRules: []string{
				"每个角色最多同时调谐3个魔法物品",
				"装备物品会自动更新角色的AC值",
				"物品转移后会解除调谐状态",
			},
			Priority:     15,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"add_item", "remove_item", "get_inventory",
				"equip_item", "unequip_item", "get_equipment",
				"transfer_item", "attune_item", "unattune_item",
				"add_currency", "use_magic_item", "recharge", "magic_item",
				"start_crafting", "advance_crafting", "complete_crafting", "get_crafting_recipes",
				"inventory", "物品", "装备", "库存", "调谐", "货币", "金币",
				"制作", "配方", "锻造", "炼金",
			},
		}, registry, llmClient),
	}
}

// ToolsForTask 根据任务描述动态过滤工具
func (a *InventoryAgent) ToolsForTask(task string) []tool.Tool {
	taskLower := strings.ToLower(task)
	allTools := a.Tools()
	result := make([]tool.Tool, 0)

	// 复合工具总是优先保留
	for _, t := range allTools {
		switch t.Name() {
		case "query_equipment", "query_spells", "query_feats":
			result = append(result, t)
		}
	}

	// 添加/移除物品
	if (strings.Contains(taskLower, "add") || strings.Contains(taskLower, "remove") || strings.Contains(taskLower, "give") ||
		strings.Contains(taskLower, "添加") || strings.Contains(taskLower, "移除") || strings.Contains(taskLower, "给予")) &&
		(strings.Contains(taskLower, "item") || strings.Contains(taskLower, "物品")) {
		for _, t := range allTools {
			switch t.Name() {
			case "add_item", "remove_item", "transfer_item":
				result = append(result, t)
			}
		}
		return result
	}

	// 装备/脱装备
	if strings.Contains(taskLower, "equip") || strings.Contains(taskLower, "unequip") || strings.Contains(taskLower, "装备") ||
		strings.Contains(taskLower, "穿戴") || strings.Contains(taskLower, "脱下") {
		for _, t := range allTools {
			switch t.Name() {
			case "equip_item", "unequip_item", "get_equipment":
				result = append(result, t)
			}
		}
		return result
	}

	// 调谐魔法物品
	if strings.Contains(taskLower, "attune") || strings.Contains(taskLower, "unattune") || strings.Contains(taskLower, "调谐") {
		for _, t := range allTools {
			switch t.Name() {
			case "attune_item", "unattune_item", "use_magic_item", "recharge_magic_items":
				result = append(result, t)
			}
		}
		return result
	}

	// 货币相关
	if strings.Contains(taskLower, "currency") || strings.Contains(taskLower, "gold") || strings.Contains(taskLower, "coin") ||
		strings.Contains(taskLower, "货币") || strings.Contains(taskLower, "金币") || strings.Contains(taskLower, "铜币") {
		for _, t := range allTools {
			switch t.Name() {
			case "add_currency", "get_inventory":
				result = append(result, t)
			}
		}
		return result
	}

	// 查询库存/装备
	if (strings.Contains(taskLower, "get") || strings.Contains(taskLower, "list") || strings.Contains(taskLower, "show") ||
		strings.Contains(taskLower, "查看") || strings.Contains(taskLower, "查询")) &&
		(strings.Contains(taskLower, "inventory") || strings.Contains(taskLower, "item") || strings.Contains(taskLower, "equipment") ||
			strings.Contains(taskLower, "库存") || strings.Contains(taskLower, "物品") || strings.Contains(taskLower, "装备")) {
		for _, t := range allTools {
			switch t.Name() {
			case "get_inventory", "get_equipment", "get_magic_item_bonus":
				result = append(result, t)
			}
		}
		return result
	}

	// 魔法物品使用
	if strings.Contains(taskLower, "magic") || strings.Contains(taskLower, "use") || strings.Contains(taskLower, "recharge") ||
		strings.Contains(taskLower, "魔法") || strings.Contains(taskLower, "使用") || strings.Contains(taskLower, "充能") {
		for _, t := range allTools {
			switch t.Name() {
			case "use_magic_item", "recharge_magic_items", "get_magic_item_bonus":
				result = append(result, t)
			}
		}
		return result
	}

	// 制作相关
	if strings.Contains(taskLower, "craft") || strings.Contains(taskLower, "crafting") || strings.Contains(taskLower, "recipe") ||
		strings.Contains(taskLower, "制作") || strings.Contains(taskLower, "锻造") || strings.Contains(taskLower, "炼金") ||
		strings.Contains(taskLower, "配方") {
		for _, t := range allTools {
			switch t.Name() {
			case "start_crafting", "advance_crafting", "complete_crafting", "get_crafting_recipes":
				result = append(result, t)
			}
		}
		return result
	}

	// 数据查询（装备/法术/专长）
	if strings.Contains(taskLower, "query") || strings.Contains(taskLower, "search") || strings.Contains(taskLower, "find") ||
		strings.Contains(taskLower, "查询") || strings.Contains(taskLower, "搜索") {
		for _, t := range allTools {
			switch t.Name() {
			case "query_equipment", "query_spells", "query_feats", "get_crafting_recipes":
				result = append(result, t)
			}
		}
		return result
	}

	// 默认返回所有工具
	if len(result) == 0 {
		return allTools
	}
	return result
}
