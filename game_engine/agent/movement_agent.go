package agent

import (
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// MovementAgent 移动管理Agent
type MovementAgent struct {
	*BaseSubAgent
}

// NewMovementAgent 创建移动管理Agent
func NewMovementAgent(registry *tool.ToolRegistry, llmClient llm.LLMClient) *MovementAgent {
	return &MovementAgent{
		BaseSubAgent: NewBaseSubAgent(SubAgentConfig{
			Name:         SubAgentNameMovement,
			Description:  "移动管理Agent，负责跳跃、跌落、窒息、遭遇检定",
			TemplateFile: "movement_system.md",
			DomainIntro:  "你是D&D 5e移动与环境专家。",
			DomainRule:   "所有移动操作必须通过调用Tools完成。",
			KeyRules: []string{
				"跳远距离=力量值（有助跑），跳高=3+力量修正",
				"跌落每10尺1d6伤害，最多20d6",
				"闭气时间=1+体质修正分钟",
			},
			Priority:     9,
			Dependencies: []string{SubAgentNameCharacter},
			Keywords: []string{
				"perform_jump", "apply_fall_damage", "calculate_breath_holding",
				"apply_suffocation", "encounter_check",
				"jump", "fall", "suffocation",
				"跳跃", "跌落", "窒息", "遭遇",
			},
		}, registry, llmClient),
	}
}
