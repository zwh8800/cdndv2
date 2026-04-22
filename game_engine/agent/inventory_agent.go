package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// InventoryAgent 库存和装备管理Agent
type InventoryAgent struct {
	*BaseSubAgent
}

// NewInventoryAgent 创建库存管理Agent
func NewInventoryAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *InventoryAgent {
	return &InventoryAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameInventory,
			Description:  "库存管理Agent，负责物品管理、装备管理、魔法物品、货币管理",
			TemplateFile: "inventory_system.md",
			DomainIntro:  "你是D&D 5e库存和装备管理专家。",
			DomainRule:   "所有库存操作必须通过调用Tools完成，不得自行计算。",
			KeyRules: []string{
				"每个角色最多同时调谐3个魔法物品",
				"装备物品会自动更新角色的AC值",
				"物品转移后会解除调谐状态",
			},
			Priority:     15,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"manage_item", "equip_item", "use_item", "query_inventory",
				"inventory", "物品", "装备", "库存", "调谐", "货币", "金币", "魔法物品",
			},
		}, registry, llmClient),
	}
}
