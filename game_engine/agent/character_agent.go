package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// CharacterAgent 角色管理Agent
type CharacterAgent struct {
	*BaseSubAgent
}

// NewCharacterAgent 创建角色管理Agent
func NewCharacterAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *CharacterAgent {
	return &CharacterAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameCharacter,
			Description:  "角色管理Agent，负责角色创建、查询、更新、经验、休息",
			TemplateFile: "character_system.md",
			DomainIntro:  "你是D&D 5e角色管理专家。",
			DomainRule:   "所有角色操作必须通过调用Tools完成，不得自行计算。",
			KeyRules:     nil,
			Priority:     10,
			Dependencies: nil,
			Keywords: []string{
				"create_character", "create_pc", "create_npc", "create_enemy", "create_companion",
				"get_actor", "get_pc", "list_actors", "update_actor", "remove_actor",
				"add_experience", "level_up", "short_rest", "long_rest",
				"character", "角色", "创建", "升级", "经验", "休息",
			},
		}, registry, llmClient),
	}
}
