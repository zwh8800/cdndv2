package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// CombatAgent 战斗管理Agent
type CombatAgent struct {
	*BaseSubAgent
}

// NewCombatAgent 创建战斗管理Agent
func NewCombatAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *CombatAgent {
	return &CombatAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameCombat,
			Description:  "战斗管理Agent，负责战斗初始化、回合管理、攻击、伤害治疗",
			TemplateFile: "combat_system.md",
			DomainIntro:  "你是D&D 5e战斗系统专家。",
			DomainRule:   "所有战斗操作必须通过调用Tools完成，不得自行计算。",
			KeyRules:     nil,
			Priority:     20,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"initiate_combat", "combat_action", "resolve_combat", "query_combat",
				"combat", "战斗", "攻击", "回合", "伤害", "治疗", "移动",
			},
		}, registry, llmClient),
	}
}
