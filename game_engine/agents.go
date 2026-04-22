package gameengine

import (
	"github.com/zwh8800/cdndv2/game_engine/agent"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
	"github.com/zwh8800/dnd-core/pkg/engine"
)

// registerAgentTools 注册主/子Agent所需工具
// 使用组合工具（composite tools）替代大量底层工具，SubAgent只看到组合工具
// MainAgent只看到 delegate_task + 只读查询组合工具 + 阶段管理工具
func registerAgentTools(registry *tool.ToolRegistry, engine *engine.Engine) {
	// ========== MainAgent 专用 ==========
	// 委托工具
	registry.Register(tool.NewDelegateTaskTool(), []string{agent.MainAgentName}, "delegation")
	// 阶段管理（MainAgent可直接操作）
	registry.Register(tool.NewSetPhaseTool(engine), []string{agent.MainAgentName, agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.SubAgentNameRules}, "phase")
	registry.Register(tool.NewGetPhaseTool(engine), []string{agent.MainAgentName, agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.SubAgentNameRules}, "phase")

	// ========== 角色管理组合工具 → CharacterAgent ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewCreatePlayerCharacterTool(engine), []string{agent.SubAgentNameCharacter}, "character")
	registry.Register(tool.NewSpawnCreatureTool(engine), []string{agent.SubAgentNameCharacter, agent.SubAgentNameCombat}, "character")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewQueryCharacterTool(engine), []string{agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.SubAgentNameRules, agent.MainAgentName}, "character")

	// ========== 战斗系统组合工具 → CombatAgent ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewInitiateCombatTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewCombatActionTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewResolveCombatTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewQueryCombatTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "combat")

	// ========== 规则检定组合工具 → RulesAgent ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewPerformCheckTool(engine), []string{agent.SubAgentNameRules}, "check")
	registry.Register(tool.NewCastSpellCompositeTool(engine), []string{agent.SubAgentNameRules}, "spell")
	registry.Register(tool.NewManageSpellsTool(engine), []string{agent.SubAgentNameRules}, "spell")
	registry.Register(tool.NewTakeRestTool(engine), []string{agent.SubAgentNameRules}, "rest")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewQuerySpellStatusTool(engine), []string{agent.SubAgentNameRules, agent.MainAgentName}, "spell")

	// ========== 库存管理组合工具 → InventoryAgent ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewManageItemTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	registry.Register(tool.NewEquipItemCompositeTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	registry.Register(tool.NewUseItemTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewQueryInventoryTool(engine), []string{agent.SubAgentNameInventory, agent.MainAgentName}, "inventory")

	// ========== 世界叙事组合工具 → WorldAgent ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewSetupSceneTool(engine), []string{agent.SubAgentNameWorld}, "world")
	registry.Register(tool.NewNPCSocialInteractionTool(engine), []string{agent.SubAgentNameWorld}, "world")
	registry.Register(tool.NewManageQuestTool(engine), []string{agent.SubAgentNameWorld}, "world")
	registry.Register(tool.NewTravelTool(engine), []string{agent.SubAgentNameWorld}, "world")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewQueryWorldTool(engine), []string{agent.SubAgentNameWorld, agent.MainAgentName}, "world")

	// ========== 数据查询组合工具 → 全局只读 ==========
	// 只读 - MainAgent + 所有需要查数据的SubAgent
	registry.Register(tool.NewLookupGameDataTool(engine), []string{
		agent.MainAgentName,
		agent.SubAgentNameCharacter,
		agent.SubAgentNameCombat,
		agent.SubAgentNameRules,
		agent.SubAgentNameWorld,
	}, "data_query")
}

// createSubAgents 创建子Agent（5个组合Agent替代原有11个）
func createSubAgents(registry *tool.ToolRegistry, llmClient llm.LLMClient) map[string]agent.SubAgent {
	return map[string]agent.SubAgent{
		agent.SubAgentNameCharacter: agent.NewCharacterAgent(registry, llmClient),
		agent.SubAgentNameCombat:    agent.NewCombatAgent(registry, llmClient),
		agent.SubAgentNameRules:     agent.NewRulesAgent(registry, llmClient),
		agent.SubAgentNameInventory: agent.NewInventoryAgent(registry, llmClient),
		agent.SubAgentNameWorld:     agent.NewWorldAgent(registry, llmClient),
	}
}

// createRouter 创建路由Agent
func createRouter(llmClient llm.LLMClient, agents map[string]agent.SubAgent) *agent.RouterAgent {
	return agent.NewRouterAgent(llmClient, agents)
}
