package agent

import (
	"strings"

	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// NarrativeAgent 叙事与场景管理Agent
type NarrativeAgent struct {
	*BaseSubAgent
}

// NewNarrativeAgent 创建叙事管理Agent
func NewNarrativeAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *NarrativeAgent {
	return &NarrativeAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameNarrative,
			Description:  "叙事管理Agent，负责场景管理、旅行探索、陷阱交互、NPC社交、任务管理、时间推进、环境移动",
			TemplateFile: "narrative_system.md",
			DomainIntro:  "你是D&D 5e叙事与场景管理专家。",
			DomainRule:   "所有场景和探索操作必须通过调用Tools完成，不得自行模拟。",
			KeyRules: []string{
				"场景中有角色时无法删除该场景",
				"旅行速度影响遭遇概率和警觉性（快速=减5感知，慢速=可隐秘行进）",
				"陷阱检测需要感知（察觉）检定，解除需要敏捷（巧手）检定",
				"NPC态度分为友好、冷淡、敌对，游说/威吓/欺瞒检定影响态度",
			},
			Priority:     10,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"create_scene", "get_scene", "update_scene", "delete_scene", "list_scenes",
				"set_current_scene", "get_current_scene",
				"add_scene_connection", "remove_scene_connection",
				"move_actor_to_scene", "get_scene_actors",
				"add_item_to_scene", "remove_item_from_scene", "get_scene_items",
				"start_travel", "advance_travel", "forage", "navigate",
				"place_trap", "detect_trap", "disarm_trap", "trigger_trap",
				"interact_with_npc", "get_npc_attitude",
				"create_quest", "accept_quest", "update_quest", "complete_quest", "fail_quest",
				"get_quest", "list_quests", "get_actor_quests",
				"set_lifestyle", "advance_game_time",
				"perform_jump", "apply_fall_damage", "apply_suffocation", "perform_encounter_check",
				"calculate_breath_holding",
				"scene", "场景", "旅行", "探索", "觅食", "导航", "陷阱", "移动", "地点",
				"npc", "NPC", "社交", "态度", "游说", "威吓", "欺瞒",
				"任务", "quest", "时间", "生活方式",
				"跳跃", "跌落", "窒息", "遭遇",
			},
		}, registry, llmClient),
	}
}

// ToolsForTask 根据任务描述动态过滤工具
func (a *NarrativeAgent) ToolsForTask(task string) []tool.Tool {
	taskLower := strings.ToLower(task)
	allTools := a.Tools()
	result := make([]tool.Tool, 0)

	// 复合工具总是优先保留
	for _, t := range allTools {
		switch t.Name() {
		case "create_connected_scene", "show_scene_detail", "move_to_scene":
			result = append(result, t)
		}
	}

	// 场景创建/修改/删除相关
	if (strings.Contains(taskLower, "create") || strings.Contains(taskLower, "new") || strings.Contains(taskLower, "update") ||
		strings.Contains(taskLower, "delete") || strings.Contains(taskLower, "remove") ||
		strings.Contains(taskLower, "创建") || strings.Contains(taskLower, "新建") || strings.Contains(taskLower, "更新") ||
		strings.Contains(taskLower, "删除")) && (strings.Contains(taskLower, "scene") ||
		strings.Contains(taskLower, "场景")) {
		for _, t := range allTools {
			switch t.Name() {
			case "create_scene", "update_scene", "delete_scene", "add_scene_connection", "remove_scene_connection",
				"set_current_scene":
				result = append(result, t)
			}
		}
		return result
	}

	// 移动到场景/切换场景
	if strings.Contains(taskLower, "move") || strings.Contains(taskLower, "travel") || strings.Contains(taskLower, "go") ||
		strings.Contains(taskLower, "移动") || strings.Contains(taskLower, "前往") || strings.Contains(taskLower, "进入") {
		for _, t := range allTools {
			switch t.Name() {
			case "move_actor_to_scene", "set_current_scene", "get_scene":
				result = append(result, t)
			}
		}
		return result
	}

	// 查询场景信息
	if (strings.Contains(taskLower, "get") || strings.Contains(taskLower, "list") || strings.Contains(taskLower, "show") ||
		strings.Contains(taskLower, "查看") || strings.Contains(taskLower, "查询") || strings.Contains(taskLower, "显示")) &&
		(strings.Contains(taskLower, "scene") || strings.Contains(taskLower, "场景")) {
		for _, t := range allTools {
			switch t.Name() {
			case "get_scene", "list_scenes", "get_current_scene", "get_scene_actors", "get_scene_items":
				result = append(result, t)
			}
		}
		return result
	}

	// 场景物品相关
	if strings.Contains(taskLower, "item") && (strings.Contains(taskLower, "scene") || strings.Contains(taskLower, "场景")) ||
		strings.Contains(taskLower, "物品") && (strings.Contains(taskLower, "场景") || strings.Contains(taskLower, "房间")) {
		for _, t := range allTools {
			switch t.Name() {
			case "add_item_to_scene", "remove_item_from_scene", "get_scene_items":
				result = append(result, t)
			}
		}
		return result
	}

	// 旅行探索相关
	if strings.Contains(taskLower, "travel") || strings.Contains(taskLower, "explore") || strings.Contains(taskLower, "forage") ||
		strings.Contains(taskLower, "navigate") || strings.Contains(taskLower, "旅行") || strings.Contains(taskLower, "探索") ||
		strings.Contains(taskLower, "觅食") || strings.Contains(taskLower, "导航") {
		for _, t := range allTools {
			switch t.Name() {
			case "start_travel", "advance_travel", "forage", "navigate", "perform_encounter_check":
				result = append(result, t)
			}
		}
		return result
	}

	// 陷阱相关
	if strings.Contains(taskLower, "trap") || strings.Contains(taskLower, "陷阱") {
		for _, t := range allTools {
			switch t.Name() {
			case "place_trap", "detect_trap", "disarm_trap", "trigger_trap":
				result = append(result, t)
			}
		}
		return result
	}

	// NPC社交相关
	if strings.Contains(taskLower, "npc") || strings.Contains(taskLower, "interact") || strings.Contains(taskLower, "social") ||
		strings.Contains(taskLower, "attitude") || strings.Contains(taskLower, "社交") || strings.Contains(taskLower, "交互") ||
		strings.Contains(taskLower, "态度") {
		for _, t := range allTools {
			switch t.Name() {
			case "interact_with_npc", "get_npc_attitude":
				result = append(result, t)
			}
		}
		return result
	}

	// 任务相关
	if strings.Contains(taskLower, "quest") || strings.Contains(taskLower, "task") || strings.Contains(taskLower, "accept") ||
		strings.Contains(taskLower, "complete") || strings.Contains(taskLower, "任务") || strings.Contains(taskLower, "接受") ||
		strings.Contains(taskLower, "完成") {
		for _, t := range allTools {
			switch t.Name() {
			case "create_quest", "accept_quest", "update_quest", "complete_quest", "fail_quest",
				"get_quest", "list_quests", "get_actor_quests":
				result = append(result, t)
			}
		}
		return result
	}

	// 时间/生活方式相关
	if strings.Contains(taskLower, "time") || strings.Contains(taskLower, "advance") || strings.Contains(taskLower, "lifestyle") ||
		strings.Contains(taskLower, "时间") || strings.Contains(taskLower, "推进") || strings.Contains(taskLower, "生活方式") {
		for _, t := range allTools {
			switch t.Name() {
			case "advance_game_time", "set_lifestyle":
				result = append(result, t)
			}
		}
		return result
	}

	// 移动/跳跃/环境伤害相关
	if strings.Contains(taskLower, "jump") || strings.Contains(taskLower, "fall") || strings.Contains(taskLower, "suffocation") ||
		strings.Contains(taskLower, "breath") || strings.Contains(taskLower, "encounter") || strings.Contains(taskLower, "跳跃") ||
		strings.Contains(taskLower, "坠落") || strings.Contains(taskLower, "窒息") || strings.Contains(taskLower, "遭遇") {
		for _, t := range allTools {
			switch t.Name() {
			case "perform_jump", "apply_fall_damage", "apply_suffocation", "calculate_breath_holding", "perform_encounter_check":
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
