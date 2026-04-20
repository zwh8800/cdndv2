package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// RulesAgent 规则仲裁Agent
type RulesAgent struct {
	*BaseSubAgent
}

// NewRulesAgent 创建规则仲裁Agent
func NewRulesAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *RulesAgent {
	return &RulesAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameRules,
			Description:  "规则仲裁Agent，负责检定、豁免、法术施放、专注管理",
			TemplateFile: "rules_system.md",
			DomainIntro:  "你是D&D 5e规则仲裁专家。",
			DomainRule:   "所有检定和法术操作必须通过调用Tools完成，不得自行计算。",
			KeyRules:     nil,
			Priority:     5,
			Dependencies: nil,
			Keywords: []string{
				"check", "save", "spell", "concentration", "skill",
				"perform_ability_check", "perform_skill_check", "perform_saving_throw",
				"cast_spell", "get_spell_slots", "concentration_check",
				"检定", "豁免", "法术", "专注", "技能",
			},
		}, registry, llmClient),
	}
}
