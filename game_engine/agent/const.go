package agent

const (
	MainAgentName         = "main_agent"
	SubAgentNameCharacter = "character_agent"
	SubAgentNameCombat    = "combat_agent"
	SubAgentNameRules     = "rules_agent"
	SubAgentNameInventory = "inventory_agent"
	SubAgentNameWorld     = "world_agent"

	// Deprecated: 以下常量保留向后兼容，新代码请使用上面的5个Agent
	SubAgentNameNarrative = "narrative_agent"
	SubAgentNameNPC       = "npc_agent"
	SubAgentNameMemory    = "memory_agent"
	SubAgentNameMovement  = "movement_agent"
	SubAgentNameMount     = "mount_agent"
	SubAgentNameCrafting  = "crafting_agent"
	SubAgentNameDataQuery = "data_query_agent"
)
