package agent

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zwh8800/dnd-core/pkg/data"

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
				"create_player_character", "spawn_creature", "query_character",
				"character", "角色", "创建", "升级", "经验",
			},
			ExtraTemplateData: func(ctx *AgentContext) map[string]any {
				return map[string]any{
					"RaceList":       formatRaceList(),
					"ClassList":      formatClassList(),
					"BackgroundList": formatBackgroundList(),
				}
			},
		}, registry, llmClient),
	}
}

// formatRaceList 格式化种族列表，包含子种族信息
func formatRaceList() string {
	names := data.GetRaceNames()
	sort.Strings(names)
	var parts []string
	for _, name := range names {
		race := data.GetRace(name)
		if race == nil {
			continue
		}
		if len(race.Subraces) > 0 {
			parts = append(parts, fmt.Sprintf("- %s（子种族：%s）", name, strings.Join(race.Subraces, "、")))
		} else {
			parts = append(parts, fmt.Sprintf("- %s", name))
		}
	}
	return strings.Join(parts, "\n")
}

// formatClassList 格式化职业列表
func formatClassList() string {
	names := data.GetClassNames()
	sort.Strings(names)
	var parts []string
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("- %s", name))
	}
	return strings.Join(parts, "\n")
}

// formatBackgroundList 格式化背景列表
func formatBackgroundList() string {
	names := data.GetBackgroundNames()
	sort.Strings(names)
	var parts []string
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("- %s", name))
	}
	return strings.Join(parts, "\n")
}
