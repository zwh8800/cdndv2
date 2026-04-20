package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// MemoryAgent 记忆管理Agent
type MemoryAgent struct {
	*BaseSubAgent
}

// NewMemoryAgent 创建记忆管理Agent
func NewMemoryAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *MemoryAgent {
	return &MemoryAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameMemory,
			Description:  "记忆管理Agent，负责任务系统、生活方式、游戏时间",
			TemplateFile: "memory_system.md",
			DomainIntro:  "你是D&D 5e任务与记忆管理专家。",
			DomainRule:   "所有任务和状态操作必须通过调用Tools完成。",
			KeyRules: []string{
				"任务有可接、进行中、已完成、已失败四种状态",
				"完成任务会发放经验和金币奖励",
				"生活方式影响每日开销",
			},
			Priority:     8,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"create_quest", "get_quest", "list_quests", "accept_quest", "complete_quest", "fail_quest",
				"quest", "任务", "生活方式", "时间", "存档",
			},
		}, registry, llmClient),
	}
}
