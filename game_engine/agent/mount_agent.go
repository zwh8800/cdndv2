package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// MountAgent 坐骑管理Agent
type MountAgent struct {
	*BaseSubAgent
}

// NewMountAgent 创建坐骑管理Agent
func NewMountAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *MountAgent {
	return &MountAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameMount,
			Description:  "坐骑管理Agent，负责骑乘、下马、坐骑速度计算",
			TemplateFile: "mount_system.md",
			DomainIntro:  "你是D&D 5e坐骑管理专家。",
			DomainRule:   "所有坐骑操作必须通过调用Tools完成。",
			KeyRules: []string{
				"骑乘需要骑手和坐骑在同一位置",
				"骑手使用坐骑的移动速度",
				"坐骑受惊可能需要驾驭检定",
			},
			Priority:     7,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"mount", "dismount", "mount_speed",
				"骑乘", "下马", "坐骑", "马",
			},
		}, registry, llmClient),
	}
}
