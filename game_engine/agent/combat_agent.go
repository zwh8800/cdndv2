package agent

import (
	"strings"

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

// ToolsForTask 根据任务描述动态过滤工具
func (a *CombatAgent) ToolsForTask(task string) []tool.Tool {
	taskLower := strings.ToLower(task)
	allTools := a.Tools()
	result := make([]tool.Tool, 0)

	// 复合工具总是优先保留
	for _, t := range allTools {
		switch t.Name() {
		case "combat_attack", "combat_start", "combat_heal", "combat_death_save", "submit_combat_plan", "show_combat_status", "query_spells":
			result = append(result, t)
		}
	}

	// 攻击伤害相关
	if strings.Contains(taskLower, "attack") || strings.Contains(taskLower, "damage") || strings.Contains(taskLower, "hit") ||
		strings.Contains(taskLower, "攻击") || strings.Contains(taskLower, "伤害") || strings.Contains(taskLower, "命中") {
		for _, t := range allTools {
			switch t.Name() {
			case "execute_attack", "execute_damage", "get_current_combat", "get_actor":
				result = append(result, t)
			}
		}
		return result
	}

	// 开始/结束战斗相关
	if strings.Contains(taskLower, "start") || strings.Contains(taskLower, "begin") || strings.Contains(taskLower, "end") ||
		strings.Contains(taskLower, "战斗") || strings.Contains(taskLower, "开始") || strings.Contains(taskLower, "结束") {
		for _, t := range allTools {
			switch t.Name() {
			case "start_combat", "start_combat_with_surprise", "end_combat", "next_turn":
				result = append(result, t)
			}
		}
		return result
	}

	// 治疗相关
	if strings.Contains(taskLower, "heal") || strings.Contains(taskLower, "cure") || strings.Contains(taskLower, "治疗") ||
		strings.Contains(taskLower, "治愈") {
		for _, t := range allTools {
			switch t.Name() {
			case "execute_healing", "get_actor":
				result = append(result, t)
			}
		}
		return result
	}

	// 死亡豁免相关
	if strings.Contains(taskLower, "death") || strings.Contains(taskLower, "save") || strings.Contains(taskLower, "稳定") ||
		strings.Contains(taskLower, "死亡") || strings.Contains(taskLower, "豁免") {
		for _, t := range allTools {
			switch t.Name() {
			case "perform_death_save", "get_actor":
				result = append(result, t)
			}
		}
		return result
	}

	// 检定相关（能力、技能、豁免）
	if strings.Contains(taskLower, "check") || strings.Contains(taskLower, "save") || strings.Contains(taskLower, "检定") ||
		strings.Contains(taskLower, "豁免") || strings.Contains(taskLower, "技能") || strings.Contains(taskLower, "能力") {
		for _, t := range allTools {
			switch t.Name() {
			case "perform_ability_check", "perform_skill_check", "perform_saving_throw", "get_passive_perception":
				result = append(result, t)
			}
		}
		return result
	}

	// 法术相关
	if strings.Contains(taskLower, "spell") || strings.Contains(taskLower, "cast") || strings.Contains(taskLower, "prepare") ||
		strings.Contains(taskLower, "concentration") || strings.Contains(taskLower, "法术") || strings.Contains(taskLower, "施法") ||
		strings.Contains(taskLower, "专注") {
		for _, t := range allTools {
			switch t.Name() {
			case "cast_spell", "prepare_spells", "learn_spell", "concentration_check", "end_concentration",
				"get_spell_slots", "list_spells", "get_spell":
				result = append(result, t)
			}
		}
		return result
	}

	// 休息相关
	if strings.Contains(taskLower, "rest") || strings.Contains(taskLower, "休息") || strings.Contains(taskLower, "长休") ||
		strings.Contains(taskLower, "短休") {
		for _, t := range allTools {
			switch t.Name() {
			case "short_rest", "start_long_rest", "end_long_rest":
				result = append(result, t)
			}
		}
		return result
	}

	// 移动相关（战斗内）
	if strings.Contains(taskLower, "move") && strings.Contains(taskLower, "combat") ||
		strings.Contains(taskLower, "移动") && strings.Contains(taskLower, "战斗") {
		for _, t := range allTools {
			switch t.Name() {
			case "move_actor", "get_current_combat":
				result = append(result, t)
			}
		}
		return result
	}

	// 查询状态
	if strings.Contains(taskLower, "status") || strings.Contains(taskLower, "current") || strings.Contains(taskLower, "get") ||
		strings.Contains(taskLower, "状态") || strings.Contains(taskLower, "当前") {
		for _, t := range allTools {
			switch t.Name() {
			case "get_current_combat", "get_current_turn", "get_actor":
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
