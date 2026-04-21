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

	// ========== 库存管理工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewAddItemTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	registry.Register(tool.NewRemoveItemTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	registry.Register(tool.NewEquipItemTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	registry.Register(tool.NewUnequipItemTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	registry.Register(tool.NewTransferItemTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	registry.Register(tool.NewAttuneItemTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	registry.Register(tool.NewAddCurrencyTool(engine), []string{agent.SubAgentNameInventory}, "inventory")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetInventoryTool(engine), []string{agent.SubAgentNameInventory, agent.MainAgentName}, "inventory")
	registry.Register(tool.NewGetEquipmentTool(engine), []string{agent.SubAgentNameInventory, agent.MainAgentName}, "inventory")

	// ========== 魔法物品工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewUseMagicItemTool(engine), []string{agent.SubAgentNameInventory}, "magic_item")
	registry.Register(tool.NewUnattuneItemTool(engine), []string{agent.SubAgentNameInventory}, "magic_item")
	registry.Register(tool.NewRechargeMagicItemsTool(engine), []string{agent.SubAgentNameInventory}, "magic_item")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetMagicItemBonusTool(engine), []string{agent.SubAgentNameInventory, agent.MainAgentName}, "magic_item")

	// ========== 叙事与场景工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewCreateSceneTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	registry.Register(tool.NewUpdateSceneTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	registry.Register(tool.NewDeleteSceneTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	registry.Register(tool.NewSetCurrentSceneTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	registry.Register(tool.NewAddSceneConnectionTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	registry.Register(tool.NewRemoveSceneConnectionTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	registry.Register(tool.NewMoveActorToSceneTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	registry.Register(tool.NewAddItemToSceneTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	registry.Register(tool.NewRemoveItemFromSceneTool(engine), []string{agent.SubAgentNameNarrative}, "scene")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetSceneTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "scene")
	registry.Register(tool.NewListScenesTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "scene")
	registry.Register(tool.NewGetCurrentSceneTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "scene")
	registry.Register(tool.NewGetSceneActorsTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "scene")
	registry.Register(tool.NewGetSceneItemsTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "scene")

	// ========== 探索工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewStartTravelTool(engine), []string{agent.SubAgentNameNarrative}, "exploration")
	registry.Register(tool.NewAdvanceTravelTool(engine), []string{agent.SubAgentNameNarrative}, "exploration")
	registry.Register(tool.NewForageTool(engine), []string{agent.SubAgentNameNarrative}, "exploration")
	registry.Register(tool.NewNavigateTool(engine), []string{agent.SubAgentNameNarrative}, "exploration")

	// ========== 陷阱工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewPlaceTrapTool(engine), []string{agent.SubAgentNameNarrative}, "trap")
	registry.Register(tool.NewDetectTrapTool(engine), []string{agent.SubAgentNameNarrative}, "trap")
	registry.Register(tool.NewDisarmTrapTool(engine), []string{agent.SubAgentNameNarrative}, "trap")
	registry.Register(tool.NewTriggerTrapTool(engine), []string{agent.SubAgentNameNarrative}, "trap")

	// ========== 社交互动工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewInteractWithNPCTool(engine), []string{agent.SubAgentNameNPC}, "social")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetNPCAttitudeTool(engine), []string{agent.SubAgentNameNPC, agent.MainAgentName}, "social")

	// ========== 任务管理工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewCreateQuestTool(engine), []string{agent.SubAgentNameMemory}, "quest")
	registry.Register(tool.NewAcceptQuestTool(engine), []string{agent.SubAgentNameMemory}, "quest")
	registry.Register(tool.NewUpdateQuestObjectiveTool(engine), []string{agent.SubAgentNameMemory}, "quest")
	registry.Register(tool.NewCompleteQuestTool(engine), []string{agent.SubAgentNameMemory}, "quest")
	registry.Register(tool.NewFailQuestTool(engine), []string{agent.SubAgentNameMemory}, "quest")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetQuestTool(engine), []string{agent.SubAgentNameMemory, agent.MainAgentName}, "quest")
	registry.Register(tool.NewListQuestsTool(engine), []string{agent.SubAgentNameMemory, agent.MainAgentName}, "quest")
	registry.Register(tool.NewGetActorQuestsTool(engine), []string{agent.SubAgentNameMemory, agent.MainAgentName}, "quest")

	// ========== 生活方式与时间工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewSetLifestyleTool(engine), []string{agent.SubAgentNameMemory}, "lifestyle")
	registry.Register(tool.NewAdvanceGameTimeTool(engine), []string{agent.SubAgentNameMemory}, "lifestyle")

	// ========== 移动工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewPerformJumpTool(engine), []string{agent.SubAgentNameMovement}, "movement")
	registry.Register(tool.NewApplyFallDamageTool(engine), []string{agent.SubAgentNameMovement}, "movement")
	registry.Register(tool.NewApplySuffocationTool(engine), []string{agent.SubAgentNameMovement}, "movement")
	registry.Register(tool.NewPerformEncounterCheckTool(engine), []string{agent.SubAgentNameMovement}, "movement")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewCalculateBreathHoldingTool(engine), []string{agent.SubAgentNameMovement, agent.MainAgentName}, "movement")

	// ========== 坐骑工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewMountCreatureTool(engine), []string{agent.SubAgentNameMount}, "mount")
	registry.Register(tool.NewDismountTool(engine), []string{agent.SubAgentNameMount}, "mount")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewCalculateMountSpeedTool(engine), []string{agent.SubAgentNameMount, agent.MainAgentName}, "mount")

	// ========== 制作工具 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewStartCraftingTool(engine), []string{agent.SubAgentNameCrafting}, "crafting")
	registry.Register(tool.NewAdvanceCraftingTool(engine), []string{agent.SubAgentNameCrafting}, "crafting")
	registry.Register(tool.NewCompleteCraftingTool(engine), []string{agent.SubAgentNameCrafting}, "crafting")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetCraftingRecipesTool(engine), []string{agent.SubAgentNameCrafting, agent.MainAgentName}, "crafting")

// ========== 游戏阶段管理工具 ==========
// 注意：set_phase 是 write_flow 类操作（流程控制），而非 write_rule 类操作（D&D规则运算）。
// 按照架构设计，write_flow 允许 MainAgent 直接调用，write_rule 必须通过 delegate_task 委派。
// 这是目前唯一允许 MainAgent 直接调用的写操作工具，因为 Phase 推进属于游戏流程管理而非规则计算。
registry.Register(tool.NewSetPhaseTool(engine), []string{agent.MainAgentName, agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.SubAgentNameRules}, "phase")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetPhaseTool(engine), []string{agent.MainAgentName, agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.SubAgentNameRules}, "phase")

	// ========== 数据查询工具 ==========
	// 只读 - Data Query Agent + MainAgent
	registry.Register(tool.NewListRacesTool(engine), []string{agent.SubAgentNameDataQuery, agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetRaceTool(engine), []string{agent.SubAgentNameDataQuery, agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListClassesTool(engine), []string{agent.SubAgentNameDataQuery, agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetClassTool(engine), []string{agent.SubAgentNameDataQuery, agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListBackgroundsTool(engine), []string{agent.SubAgentNameDataQuery, agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetBackgroundTool(engine), []string{agent.SubAgentNameDataQuery, agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListMonstersTool(engine), []string{agent.SubAgentNameDataQuery, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetMonsterTool(engine), []string{agent.SubAgentNameDataQuery, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListSpellsTool(engine), []string{agent.SubAgentNameDataQuery, agent.SubAgentNameRules, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetSpellTool(engine), []string{agent.SubAgentNameDataQuery, agent.SubAgentNameRules, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListWeaponsTool(engine), []string{agent.SubAgentNameDataQuery, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListArmorsTool(engine), []string{agent.SubAgentNameDataQuery, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListMagicItemsTool(engine), []string{agent.SubAgentNameDataQuery, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListFeatsDataTool(engine), []string{agent.SubAgentNameDataQuery, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetFeatDataTool(engine), []string{agent.SubAgentNameDataQuery, agent.MainAgentName}, "data_query")
}

// createSubAgents 创建子Agent
func createSubAgents(registry *tool.ToolRegistry, llmClient llm.LLMClient) map[string]agent.SubAgent {
	// 创建并返回子Agent
	return map[string]agent.SubAgent{
		agent.SubAgentNameCharacter: agent.NewCharacterAgent(registry, llmClient),
		agent.SubAgentNameCombat:    agent.NewCombatAgent(registry, llmClient),
		agent.SubAgentNameRules:     agent.NewRulesAgent(registry, llmClient),
		agent.SubAgentNameInventory: agent.NewInventoryAgent(registry, llmClient),
		agent.SubAgentNameNarrative: agent.NewNarrativeAgent(registry, llmClient),
		agent.SubAgentNameNPC:       agent.NewNPCAgent(registry, llmClient),
		agent.SubAgentNameMemory:    agent.NewMemoryAgent(registry, llmClient),
		agent.SubAgentNameMovement:  agent.NewMovementAgent(registry, llmClient),
		agent.SubAgentNameMount:     agent.NewMountAgent(registry, llmClient),
		agent.SubAgentNameCrafting:  agent.NewCraftingAgent(registry, llmClient),
		agent.SubAgentNameDataQuery: agent.NewDataQueryAgent(registry, llmClient),
	}
}

// createRouter 创建路由Agent
func createRouter(llmClient llm.LLMClient, agents map[string]agent.SubAgent) *agent.RouterAgent {
	return agent.NewRouterAgent(llmClient, agents)
}
