package game_summary

import (
	"fmt"
	"strings"
)

// FormatForLLM 将游戏摘要格式化为LLM可读文本
func FormatForLLM(summary *GameSummary) string {
	if summary == nil {
		return "No game state available"
	}

	var parts []string

	// 游戏基本信息
	parts = append(parts, fmt.Sprintf("## 游戏状态"))
	parts = append(parts, fmt.Sprintf("- 游戏: %s", summary.GameName))
	parts = append(parts, fmt.Sprintf("- 当前阶段: %s", summary.Phase))

	// 当前场景
	if summary.CurrentScene != nil {
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("### 当前场景"))
		parts = append(parts, fmt.Sprintf("- 名称: %s", summary.CurrentScene.Name))
		if summary.CurrentScene.Description != "" {
			parts = append(parts, fmt.Sprintf("- 描述: %s", summary.CurrentScene.Description))
		}
	}

	// 玩家角色
	if summary.Player != nil {
		p := summary.Player
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("### 玩家角色"))
		parts = append(parts, fmt.Sprintf("- 名称: %s", p.Name))
		if p.Race != "" || p.Background != "" {
			raceBg := p.Race
			if p.Background != "" {
				raceBg += " " + p.Background
			}
			parts = append(parts, fmt.Sprintf("- 种族/背景: %s", raceBg))
		}
		if p.TotalLevel > 0 {
			parts = append(parts, fmt.Sprintf("- 等级: %d (经验: %d)", p.TotalLevel, p.Experience))
		}
		parts = append(parts, fmt.Sprintf("- 生命值: %d/%d", p.HitPoints, p.MaxHP))
		if p.TempHitPoints > 0 {
			parts = append(parts, fmt.Sprintf("- 临时HP: %d", p.TempHitPoints))
		}
		parts = append(parts, fmt.Sprintf("- 护甲等级: %d", p.ArmorClass))
		if p.Speed > 0 {
			parts = append(parts, fmt.Sprintf("- 速度: %d尺", p.Speed))
		}
		if p.ProficiencyBonus > 0 {
			parts = append(parts, fmt.Sprintf("- 熟练加值: +%d", p.ProficiencyBonus))
		}
		if p.AbilityScores != nil {
			a := p.AbilityScores
			parts = append(parts, fmt.Sprintf("- 属性: 力量%d 敏捷%d 体质%d 智力%d 感知%d 魅力%d",
				a.STR, a.DEX, a.CON, a.INT, a.WIS, a.CHA))
		}
		if p.Inspiration {
			parts = append(parts, "- 灵感: 有")
		}
		if len(p.Conditions) > 0 {
			parts = append(parts, fmt.Sprintf("- 状态效果: %s", strings.Join(p.Conditions, ", ")))
		}
		if p.Exhaustion > 0 {
			parts = append(parts, fmt.Sprintf("- 力竭等级: %d", p.Exhaustion))
		}
	}

	// 战斗状态
	if summary.Combat != nil {
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("### 战斗状态"))
		parts = append(parts, fmt.Sprintf("- 回合: %d", summary.Combat.Round))
		parts = append(parts, fmt.Sprintf("- 当前回合: %s", summary.Combat.TurnActorName))
	}

	// 任务状态
	if len(summary.ActiveQuests) > 0 {
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("### 活跃任务"))
		for _, q := range summary.ActiveQuests {
			parts = append(parts, fmt.Sprintf("- %s: %s", q.Title, q.Description))
		}
	}

	return strings.Join(parts, "\n")
}

// FormatCombatSummary 格式化战斗摘要
func FormatCombatSummary(combat *CombatSummary) string {
	if combat == nil {
		return "无战斗"
	}

	return fmt.Sprintf("战斗状态: %s, 回合: %d, 当前回合: %s",
		combat.Status, combat.Round, combat.TurnActorName)
}

// FormatActorSheet 格式化角色信息卡
func FormatActorSheet(actor *ActorSummary) string {
	if actor == nil {
		return "无角色信息"
	}

	return fmt.Sprintf("角色: %s (%s)\n生命值: %d/%d\n护甲等级: %d",
		actor.Name, actor.Type, actor.HitPoints, actor.MaxHP, actor.ArmorClass)
}

// FormatSceneSummary 格式化场景信息
func FormatSceneSummary(scene *SceneSummary) string {
	if scene == nil {
		return "无场景信息"
	}

	result := fmt.Sprintf("场景: %s", scene.Name)
	if scene.Description != "" {
		result += fmt.Sprintf("\n描述: %s", scene.Description)
	}
	return result
}

// FormatQuestSummary 格式化任务信息
func FormatQuestSummary(quest *QuestSummary) string {
	if quest == nil {
		return "无任务信息"
	}

	return fmt.Sprintf("任务: %s [%s]\n描述: %s",
		quest.Title, quest.Status, quest.Description)
}

// FormatQuestsList 格式化任务列表
func FormatQuestsList(quests []QuestSummary) string {
	if len(quests) == 0 {
		return "无活跃任务"
	}

	var parts []string
	for _, q := range quests {
		parts = append(parts, FormatQuestSummary(&q))
	}
	return strings.Join(parts, "\n\n")
}
