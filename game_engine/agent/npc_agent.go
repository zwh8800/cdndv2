package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// NPCAgent NPC管理Agent
type NPCAgent struct {
	*BaseSubAgent
}

// NewNPCAgent 创建NPC管理Agent
func NewNPCAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *NPCAgent {
	return &NPCAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameNPC,
			Description:  "NPC管理Agent，负责社交互动、NPC态度管理",
			TemplateFile: "npc_system.md",
			DomainIntro:  "你是D&D 5e NPC与社交互动专家。",
			DomainRule:   "所有社交操作必须通过调用Tools完成，不得自行模拟。",
			KeyRules: []string{
				"NPC态度分为友好、冷淡、敌对",
				"游说、威吓、欺瞒检定影响NPC态度",
				"NPC态度影响交易和任务获取",
			},
			Priority:     12,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"interact_with_npc", "get_npc_attitude",
				"npc", "NPC", "社交", "态度", "游说", "威吓", "欺瞒",
			},
		}, registry, llmClient),
	}
}
