package gameengine

import (
	"github.com/zwh8800/cdndv2/game_engine/agent"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
	"github.com/zwh8800/dnd-core/pkg/engine"
)

// registerAgentTools 注册主/子Agent所需工具
func registerAgentTools(registry *tool.ToolRegistry, engine *engine.Engine) {
	// 注册委托任务工具（MainAgent专用，用于委托SubAgent）
	registry.Register(tool.NewDelegateTaskTool(), []string{agent.MainAgentName}, "delegation")

	// ========== 角色管理工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewCreatePCTool(engine), []string{agent.SubAgentNameCharacter}, "character")
	registry.Register(tool.NewCreateNPCTool(engine), []string{agent.SubAgentNameCharacter}, "character")
	registry.Register(tool.NewCreateEnemyTool(engine), []string{agent.SubAgentNameCharacter, agent.SubAgentNameCombat}, "character")
	registry.Register(tool.NewCreateCompanionTool(engine), []string{agent.SubAgentNameCharacter}, "character")
	registry.Register(tool.NewUpdateActorTool(engine), []string{agent.SubAgentNameCharacter}, "character")
	registry.Register(tool.NewRemoveActorTool(engine), []string{agent.SubAgentNameCharacter}, "character")
	registry.Register(tool.NewAddExperienceTool(engine), []string{agent.SubAgentNameCharacter}, "character")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetActorTool(engine), []string{agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.SubAgentNameRules, agent.MainAgentName}, "character")
	registry.Register(tool.NewGetPCTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "character")
	registry.Register(tool.NewListActorsTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "character")

	// ========== 战斗系统工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewStartCombatTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewStartCombatWithSurpriseTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewNextTurnTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewExecuteActionTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewExecuteAttackTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewMoveActorTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewExecuteDamageTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewExecuteHealingTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewPerformDeathSaveTool(engine), []string{agent.SubAgentNameCombat, agent.SubAgentNameRules}, "combat")
	registry.Register(tool.NewEndCombatTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetCurrentCombatTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "combat")
	registry.Register(tool.NewGetCurrentTurnTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "combat")

	// ========== 规则检定工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewPerformAbilityCheckTool(engine), []string{agent.SubAgentNameRules}, "check")
	registry.Register(tool.NewPerformSkillCheckTool(engine), []string{agent.SubAgentNameRules}, "check")
	registry.Register(tool.NewPerformSavingThrowTool(engine), []string{agent.SubAgentNameRules}, "check")
	registry.Register(tool.NewShortRestTool(engine), []string{agent.SubAgentNameRules}, "rest")
	registry.Register(tool.NewCastSpellTool(engine), []string{agent.SubAgentNameRules}, "spell")
	registry.Register(tool.NewPrepareSpellsTool(engine), []string{agent.SubAgentNameRules}, "spell")
	registry.Register(tool.NewLearnSpellTool(engine), []string{agent.SubAgentNameRules}, "spell")
	registry.Register(tool.NewConcentrationCheckTool(engine), []string{agent.SubAgentNameRules}, "spell")
	registry.Register(tool.NewEndConcentrationTool(engine), []string{agent.SubAgentNameRules}, "spell")
	registry.Register(tool.NewStartLongRestTool(engine), []string{agent.SubAgentNameRules}, "rest")
	registry.Register(tool.NewEndLongRestTool(engine), []string{agent.SubAgentNameRules}, "rest")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetPassivePerceptionTool(engine), []string{agent.SubAgentNameRules, agent.MainAgentName}, "check")
	registry.Register(tool.NewGetSpellSlotsTool(engine), []string{agent.SubAgentNameRules, agent.MainAgentName}, "spell")
}

// createSubAgents 创建子Agent
func createSubAgents(registry *tool.ToolRegistry, llmClient llm.LLMClient) map[string]agent.SubAgent {
	// 创建并返回子Agent
	return map[string]agent.SubAgent{
		agent.SubAgentNameCharacter: agent.NewCharacterAgent(registry, llmClient),
		agent.SubAgentNameCombat:    agent.NewCombatAgent(registry, llmClient),
		agent.SubAgentNameRules:     agent.NewRulesAgent(registry, llmClient),
	}
}

// createRouter 创建路由Agent
func createRouter(llmClient llm.LLMClient, agents map[string]agent.SubAgent) *agent.RouterAgent {
	return agent.NewRouterAgent(llmClient, agents)
}
