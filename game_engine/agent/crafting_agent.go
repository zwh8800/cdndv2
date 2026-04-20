package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// CraftingAgent 制作管理Agent
type CraftingAgent struct {
	*BaseSubAgent
}

// NewCraftingAgent 创建制作管理Agent
func NewCraftingAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *CraftingAgent {
	return &CraftingAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameCrafting,
			Description:  "制作管理Agent，负责物品制作、配方查询",
			TemplateFile: "crafting_system.md",
			DomainIntro:  "你是D&D 5e制作系统专家。",
			DomainRule:   "所有制作操作必须通过调用Tools完成。",
			KeyRules: []string{
				"制作需要相应工具熟练度",
				"制作需要消耗金币购买材料",
				"制作进度按天数推进，有熟练可缩短时间",
			},
			Priority:     6,
			Dependencies: []string{SubAgentNameCharacter, SubAgentNameInventory},
			Keywords: []string{
				"start_crafting", "advance_crafting", "complete_crafting", "crafting_recipes",
				"craft", "制作", "配方", "锻造", "炼金",
			},
		}, registry, llmClient),
	}
}
