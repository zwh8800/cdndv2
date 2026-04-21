package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// CombatAgent 战斗与规则管理Agent
type CombatAgent struct {
	*BaseSubAgent
}

// NewCombatAgent 创建战斗管理Agent
func NewCombatAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *CombatAgent {
	return &CombatAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameCombat,
			Description:  "战斗与规则管理Agent，负责战斗初始化、回合管理、攻击、伤害治疗、检定、豁免、法术施放、专注管理、休息",
			TemplateFile: "combat_system.md",
			DomainIntro:  "你是D&D 5e战斗与规则专家。",
			DomainRule:   "所有战斗和规则操作必须通过调用Tools完成，不得自行计算。",
			KeyRules:     nil,
			Priority:     20,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"attack", "combat", "turn", "damage", "heal", "move",
				"start_combat", "end_combat", "next_turn", "execute_attack",
				"check", "save", "spell", "concentration", "skill",
				"perform_ability_check", "perform_skill_check", "perform_saving_throw",
				"cast_spell", "get_spell_slots", "concentration_check",
				"short_rest", "long_rest",
				"战斗", "攻击", "回合", "伤害", "治疗", "移动",
				"检定", "豁免", "法术", "专注", "技能", "休息",
			},
		}, registry, llmClient),
	}
}
