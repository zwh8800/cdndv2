package agent

import (
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
