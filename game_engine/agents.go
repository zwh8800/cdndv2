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
	registry.Register(tool.NewGetActorTool(engine), []string{agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.MainAgentName, agent.CombatDMAgentName}, "character")
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
	registry.Register(tool.NewPerformDeathSaveTool(engine), []string{agent.SubAgentNameCombat}, "combat")
	registry.Register(tool.NewEndCombatTool(engine), []string{agent.SubAgentNameCombat, agent.CombatDMAgentName}, "combat")
	// 只读 - MainAgent + SubAgent + CombatDMAgent
	registry.Register(tool.NewGetCurrentCombatTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName, agent.CombatDMAgentName}, "combat")
	registry.Register(tool.NewGetCurrentTurnTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName, agent.CombatDMAgentName}, "combat")

	// ========== CombatDM Agent 专用工具 ==========
	// 这些工具封装了 dnd-core 组合型战斗API，供 CombatDMAgent 的 LLM 自主调用
	registry.Register(tool.NewExecuteTurnActionTool(engine), []string{agent.CombatDMAgentName}, "combat_dm")
	registry.Register(tool.NewNextTurnWithActionsTool(engine), []string{agent.CombatDMAgentName}, "combat_dm")

	// ========== 规则检定工具（合并到 combat_agent） ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewPerformAbilityCheckTool(engine), []string{agent.SubAgentNameCombat}, "check")
	registry.Register(tool.NewPerformSkillCheckTool(engine), []string{agent.SubAgentNameCombat}, "check")
	registry.Register(tool.NewPerformSavingThrowTool(engine), []string{agent.SubAgentNameCombat}, "check")
	registry.Register(tool.NewShortRestTool(engine), []string{agent.SubAgentNameCombat}, "rest")
	registry.Register(tool.NewCastSpellTool(engine), []string{agent.SubAgentNameCombat}, "spell")
	registry.Register(tool.NewPrepareSpellsTool(engine), []string{agent.SubAgentNameCombat}, "spell")
	registry.Register(tool.NewLearnSpellTool(engine), []string{agent.SubAgentNameCombat}, "spell")
	registry.Register(tool.NewConcentrationCheckTool(engine), []string{agent.SubAgentNameCombat}, "spell")
	registry.Register(tool.NewEndConcentrationTool(engine), []string{agent.SubAgentNameCombat}, "spell")
	registry.Register(tool.NewStartLongRestTool(engine), []string{agent.SubAgentNameCombat}, "rest")
	registry.Register(tool.NewEndLongRestTool(engine), []string{agent.SubAgentNameCombat}, "rest")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetPassivePerceptionTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "check")
	registry.Register(tool.NewGetSpellSlotsTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "spell")

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

	// ========== 社交互动工具（合并到 narrative_agent） ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewInteractWithNPCTool(engine), []string{agent.SubAgentNameNarrative}, "social")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetNPCAttitudeTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "social")

	// ========== 任务管理工具（合并到 narrative_agent） ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewCreateQuestTool(engine), []string{agent.SubAgentNameNarrative}, "quest")
	registry.Register(tool.NewAcceptQuestTool(engine), []string{agent.SubAgentNameNarrative}, "quest")
	registry.Register(tool.NewUpdateQuestObjectiveTool(engine), []string{agent.SubAgentNameNarrative}, "quest")
	registry.Register(tool.NewCompleteQuestTool(engine), []string{agent.SubAgentNameNarrative}, "quest")
	registry.Register(tool.NewFailQuestTool(engine), []string{agent.SubAgentNameNarrative}, "quest")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetQuestTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "quest")
	registry.Register(tool.NewListQuestsTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "quest")
	registry.Register(tool.NewGetActorQuestsTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "quest")

	// ========== 生活方式与时间工具（合并到 narrative_agent） ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewSetLifestyleTool(engine), []string{agent.SubAgentNameNarrative}, "lifestyle")
	registry.Register(tool.NewAdvanceGameTimeTool(engine), []string{agent.SubAgentNameNarrative}, "lifestyle")

	// ========== 移动工具（合并到 narrative_agent） ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewPerformJumpTool(engine), []string{agent.SubAgentNameNarrative}, "movement")
	registry.Register(tool.NewApplyFallDamageTool(engine), []string{agent.SubAgentNameNarrative}, "movement")
	registry.Register(tool.NewApplySuffocationTool(engine), []string{agent.SubAgentNameNarrative}, "movement")
	registry.Register(tool.NewPerformEncounterCheckTool(engine), []string{agent.SubAgentNameNarrative}, "movement")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewCalculateBreathHoldingTool(engine), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "movement")

	// ========== 坐骑工具（合并到 character_agent） ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewMountCreatureTool(engine), []string{agent.SubAgentNameCharacter}, "mount")
	registry.Register(tool.NewDismountTool(engine), []string{agent.SubAgentNameCharacter}, "mount")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewCalculateMountSpeedTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "mount")

	// ========== 制作工具（合并到 inventory_agent） ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewStartCraftingTool(engine), []string{agent.SubAgentNameInventory}, "crafting")
	registry.Register(tool.NewAdvanceCraftingTool(engine), []string{agent.SubAgentNameInventory}, "crafting")
	registry.Register(tool.NewCompleteCraftingTool(engine), []string{agent.SubAgentNameInventory}, "crafting")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetCraftingRecipesTool(engine), []string{agent.SubAgentNameInventory, agent.MainAgentName}, "crafting")

	// ========== 游戏阶段管理工具 ==========
	// 注意：set_phase 是 write_flow 类操作（流程控制），而非 write_rule 类操作（D&D规则运算）。
	// 按照架构设计，write_flow 允许 MainAgent 直接调用，write_rule 必须通过 delegate_task 委派。
	// 这是目前唯一允许 MainAgent 直接调用的写操作工具，因为 Phase 推进属于游戏流程管理而非规则计算。
	registry.Register(tool.NewSetPhaseTool(engine), []string{agent.MainAgentName, agent.SubAgentNameCharacter, agent.SubAgentNameCombat}, "phase")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewGetPhaseTool(engine), []string{agent.MainAgentName, agent.SubAgentNameCharacter, agent.SubAgentNameCombat}, "phase")

	// ========== 数据查询工具（删除 data_query_agent，仅保留 character 和 main_agent） ==========
	// 只读 - character_agent + MainAgent
	registry.Register(tool.NewListRacesTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetRaceTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListClassesTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetClassTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListBackgroundsTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetBackgroundTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListMonstersTool(engine), []string{agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetMonsterTool(engine), []string{agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListSpellsTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetSpellTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListWeaponsTool(engine), []string{agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListArmorsTool(engine), []string{agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListMagicItemsTool(engine), []string{agent.MainAgentName}, "data_query")
	registry.Register(tool.NewListFeatsDataTool(engine), []string{agent.MainAgentName}, "data_query")
	registry.Register(tool.NewGetFeatDataTool(engine), []string{agent.MainAgentName}, "data_query")

	// ========== 复合工具 - 战斗系统 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewCombatAttackTool(engine, registry), []string{agent.SubAgentNameCombat}, "composite_combat")
	registry.Register(tool.NewCombatStartTool(engine, registry), []string{agent.SubAgentNameCombat}, "composite_combat")
	registry.Register(tool.NewCombatHealTool(engine, registry), []string{agent.SubAgentNameCombat}, "composite_combat")
	registry.Register(tool.NewCombatDeathSaveTool(engine, registry), []string{agent.SubAgentNameCombat}, "composite_combat")
	// 只读 - MainAgent + SubAgent
	registry.Register(tool.NewShowCombatStatusTool(engine, registry), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "composite_combat")

	// ========== 复合工具 - 角色创建 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewCreateCharacterTool(engine, registry), []string{agent.SubAgentNameCharacter}, "composite_character")
	// 只读 - character_agent + MainAgent
	registry.Register(tool.NewQueryRacesTool(engine, registry), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "composite_character")
	registry.Register(tool.NewQueryClassesTool(engine, registry), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "composite_character")
	registry.Register(tool.NewQueryBackgroundsTool(engine, registry), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "composite_character")

	// ========== 复合工具 - 场景系统 ==========
	// 写操作 - 仅 SubAgent
	registry.Register(tool.NewCreateConnectedSceneTool(engine, registry), []string{agent.SubAgentNameNarrative}, "composite_scene")
	registry.Register(tool.NewMoveToSceneTool(engine, registry), []string{agent.SubAgentNameNarrative}, "composite_scene")
	// 只读 - narrative_agent + MainAgent
	registry.Register(tool.NewShowSceneDetailTool(engine, registry), []string{agent.SubAgentNameNarrative, agent.MainAgentName}, "composite_scene")

	// ========== 复合工具 - 数据查询 ==========
	// 只读 - MainAgent + 对应 SubAgent
	registry.Register(tool.NewQuerySpellsTool(engine, registry), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "composite_query")
	registry.Register(tool.NewQueryEquipmentTool(engine, registry), []string{agent.SubAgentNameInventory, agent.MainAgentName}, "composite_query")
	registry.Register(tool.NewQueryMonstersTool(engine, registry), []string{agent.MainAgentName}, "composite_query")
	registry.Register(tool.NewQueryFeatsTool(engine, registry), []string{agent.MainAgentName}, "composite_query")
}

// createSubAgents 创建子Agent
func createSubAgents(registry *tool.ToolRegistry, llmClient llm.LLMClient) map[string]agent.SubAgent {
	// 创建并返回子Agent（11 → 4 合并后）
	return map[string]agent.SubAgent{
		agent.SubAgentNameCharacter: agent.NewCharacterAgent(registry, llmClient),
		agent.SubAgentNameCombat:    agent.NewCombatAgent(registry, llmClient),
		agent.SubAgentNameNarrative: agent.NewNarrativeAgent(registry, llmClient),
		agent.SubAgentNameInventory: agent.NewInventoryAgent(registry, llmClient),
	}
}

// createRouter 创建路由Agent
func createRouter(llmClient llm.LLMClient, agents map[string]agent.SubAgent) *agent.RouterAgent {
	return agent.NewRouterAgent(llmClient, agents)
}
