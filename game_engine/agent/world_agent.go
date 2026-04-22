package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// WorldAgent 世界与叙事管理Agent
// 合并了原 narrative/npc/memory/movement/mount/crafting/data_query 七个 Agent 的职责
type WorldAgent struct {
	*BaseSubAgent
}

// NewWorldAgent 创建世界管理Agent
func NewWorldAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *WorldAgent {
	return &WorldAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameWorld,
			Description:  "世界管理Agent，负责场景搭建、NPC社交、任务管理、旅行探索、制作等",
			TemplateFile: "world_system.md",
			DomainIntro:  "你是D&D 5e世界管理专家，负责构建和管理游戏世界的所有非战斗、非规则检定方面。",
			DomainRule:   "所有世界操作必须通过调用Tools完成，不得自行模拟结果。",
			KeyRules: []string{
				"场景搭建使用setup_scene一站式完成（创建+设为当前+放置角色/物品+建立连接）",
				"NPC社交互动使用npc_social_interaction，自动获取角色属性进行检定",
				"任务管理使用manage_quest统一入口（create/accept/update_objective/complete/fail）",
				"旅行使用travel工具完成完整流程（开始+推进+遭遇检定）",
				"旅行速度影响遭遇概率和警觉性（快速=减5感知，慢速=可隐秘行进）",
				"陷阱检测需要感知（察觉）检定，解除需要敏捷（巧手）检定",
			},
			Priority:     10,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"setup_scene", "npc_social_interaction", "manage_quest", "travel",
				"场景", "NPC", "社交", "任务", "旅行", "探索", "觅食", "导航",
				"陷阱", "地点", "制作", "工艺", "坐骑", "骑乘",
				"互动", "对话", "说服", "威吓", "欺骗", "表演",
				"quest", "scene", "travel", "trap", "craft",
			},
		}, registry, llmClient),
	}
}
