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
			Description:  "角色管理Agent，负责角色创建、查询、更新、经验、休息、骑乘",
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
				"mount", "dismount", "mount_speed",
				"character", "角色", "创建", "升级", "经验", "休息", "骑乘", "下马", "坐骑",
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

// ToolsForTask 根据任务描述动态过滤工具
func (a *CharacterAgent) ToolsForTask(task string) []tool.Tool {
	taskLower := strings.ToLower(task)
	allTools := a.Tools()
	result := make([]tool.Tool, 0)

	// 复合工具总是优先保留
	for _, t := range allTools {
		switch t.Name() {
		case "create_character", "query_races", "query_classes", "query_backgrounds":
			result = append(result, t)
		}
	}

	// 角色创建相关
	if strings.Contains(taskLower, "create") || strings.Contains(taskLower, "new") || strings.Contains(taskLower, "创建") {
		for _, t := range allTools {
			switch t.Name() {
			case "create_pc", "create_npc", "create_enemy", "create_companion":
				result = append(result, t)
			}
		}
		return result
	}

	// 查询相关
	if strings.Contains(taskLower, "query") || strings.Contains(taskLower, "list") || strings.Contains(taskLower, "get") ||
		strings.Contains(taskLower, "查询") || strings.Contains(taskLower, "列表") || strings.Contains(taskLower, "查看") {
		for _, t := range allTools {
			switch t.Name() {
			case "get_actor", "get_pc", "list_actors", "list_races", "list_classes", "list_backgrounds",
				"get_race", "get_class", "get_background":
				result = append(result, t)
			}
		}
		return result
	}

	// 经验升级相关
	if strings.Contains(taskLower, "experience") || strings.Contains(taskLower, "level") || strings.Contains(taskLower, "升级") ||
		strings.Contains(taskLower, "经验") {
		for _, t := range allTools {
			switch t.Name() {
			case "add_experience", "update_actor", "get_actor":
				result = append(result, t)
			}
		}
		return result
	}

	// 修改更新相关
	if strings.Contains(taskLower, "update") || strings.Contains(taskLower, "change") || strings.Contains(taskLower, "remove") ||
		strings.Contains(taskLower, "修改") || strings.Contains(taskLower, "更新") || strings.Contains(taskLower, "删除") {
		for _, t := range allTools {
			switch t.Name() {
			case "update_actor", "remove_actor":
				result = append(result, t)
			}
		}
		return result
	}

	// 骑乘相关
	if strings.Contains(taskLower, "mount") || strings.Contains(taskLower, "dismount") || strings.Contains(taskLower, "骑乘") ||
		strings.Contains(taskLower, "坐骑") {
		for _, t := range allTools {
			switch t.Name() {
			case "mount_creature", "dismount", "calculate_mount_speed":
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
