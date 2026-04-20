package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// DataQueryAgent 数据查询Agent
type DataQueryAgent struct {
	*BaseSubAgent
}

// NewDataQueryAgent 创建数据查询Agent
func NewDataQueryAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *DataQueryAgent {
	return &DataQueryAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameDataQuery,
			Description:  "数据查询Agent，负责查询D&D规则数据（种族、职业、背景、怪物、法术、武器等）",
			TemplateFile: "data_query_system.md",
			DomainIntro:  "你是D&D 5e规则数据查询专家。",
			DomainRule:   "所有数据查询必须通过调用Tools完成，你需要根据查询返回准确的游戏规则数据。",
			KeyRules: []string{
				"数据查询为只读操作，不修改游戏状态",
				"支持分页查询大量数据",
				"查询结果用于辅助其他Agent做出决策",
			},
			Priority:     5,
			Dependencies: []string{},
			Keywords: []string{
				"list_races", "get_race", "list_classes", "get_class",
				"list_monsters", "get_monster", "list_spells", "get_spell",
				"list_weapons", "list_armors", "list_magic_items", "list_feats",
				"种族", "职业", "背景", "怪物", "法术", "武器", "护甲", "专长", "查询",
			},
		}, registry, llmClient),
	}
}
