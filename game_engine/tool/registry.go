package tool

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/zwh8800/cdndv2/game_engine/llm"
)

// ToolRegistry Tool注册中心
type ToolRegistry struct {
	tools    map[string]Tool
	byAgent  map[string][]string // agent -> tool names
	category map[string][]string // category -> tool names
	logger   *zap.Logger
}

// SetLogger 设置日志器
func (r *ToolRegistry) SetLogger(log *zap.Logger) {
	if log != nil {
		r.logger = log
	}
}

// getLogger 获取日志器
func (r *ToolRegistry) getLogger() *zap.Logger {
	if r.logger == nil {
		r.logger = zap.NewNop()
	}
	return r.logger
}

// NewToolRegistry 创建新的Tool注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:    make(map[string]Tool),
		byAgent:  make(map[string][]string),
		category: make(map[string][]string),
	}
}

// Register 注册Tool
func (r *ToolRegistry) Register(tool Tool, agents []string, category string) {
	r.tools[tool.Name()] = tool

	for _, agent := range agents {
		r.byAgent[agent] = append(r.byAgent[agent], tool.Name())
	}

	if category != "" {
		r.category[category] = append(r.category[category], tool.Name())
	}
}

// Get 获取Tool
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// GetByAgent 获取Agent可用的Tools
func (r *ToolRegistry) GetByAgent(agent string) []Tool {
	names, ok := r.byAgent[agent]
	if !ok {
		return nil
	}

	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		if tool, ok := r.tools[name]; ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetByCategory 获取分类下的所有Tools
func (r *ToolRegistry) GetByCategory(category string) []Tool {
	names, ok := r.category[category]
	if !ok {
		return nil
	}

	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		if tool, ok := r.tools[name]; ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetAll 获取所有Tools的Schema（LLM函数调用格式）
func (r *ToolRegistry) GetAll() []map[string]any {
	schemas := make([]map[string]any, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.ParametersSchema(),
			},
		})
	}
	return schemas
}

// GetAllNames 获取所有Tool名称
func (r *ToolRegistry) GetAllNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetAllTools 获取所有Tool实例
func (r *ToolRegistry) GetAllTools() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// GetReadOnlyTools 获取所有只读工具
func (r *ToolRegistry) GetReadOnlyTools() []Tool {
	var tools []Tool
	for _, t := range r.tools {
		if t.ReadOnly() {
			tools = append(tools, t)
		}
	}
	return tools
}

// GetReadOnlySchemas 获取只读工具的 LLM 函数调用格式 Schema
func (r *ToolRegistry) GetReadOnlySchemas() []map[string]any {
	schemas := make([]map[string]any, 0)
	for _, t := range r.tools {
		if t.ReadOnly() {
			schemas = append(schemas, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name(),
					"description": t.Description(),
					"parameters":  t.ParametersSchema(),
				},
			})
		}
	}
	return schemas
}

// GetAgentsForTool 获取工具所属的Agent列表
func (r *ToolRegistry) GetAgentsForTool(toolName string) []string {
	var agents []string
	for agent, tools := range r.byAgent {
		for _, t := range tools {
			if t == toolName {
				agents = append(agents, agent)
				break
			}
		}
	}
	return agents
}

// IsToolAllowedForAgent 判断指定 Agent 是否有权调用该工具。
func (r *ToolRegistry) IsToolAllowedForAgent(agentName, toolName string) bool {
	for _, name := range r.byAgent[agentName] {
		if name == toolName {
			return true
		}
	}
	return false
}

// IsToolReadOnly 判断指定工具是否为只读工具
// 实现 llm.ToolReadOnlyChecker 接口，供 ContextCompressor 使用
// 返回 (是否只读, 是否找到该工具)
func (r *ToolRegistry) IsToolReadOnly(toolName string) (bool, bool) {
	t, ok := r.tools[toolName]
	if !ok {
		return false, false
	}
	return t.ReadOnly(), true
}

// GetReadOnlySchemasByPhase 根据游戏阶段返回只读工具的 LLM 函数调用格式 Schema
// 按阶段裁剪工具集，减少 LLM 每次调用时的工具数量，提高选择准确率
// phase 可选值: character_creation, exploration, combat, rest
func (r *ToolRegistry) GetReadOnlySchemasByPhase(phase string) []map[string]any {
	baseTools := r.getPhaseBaseTools()
	phaseTools := r.getPhaseSpecificTools(phase)

	toolSet := make(map[string]bool)
	for _, t := range baseTools {
		toolSet[t] = true
	}
	for _, t := range phaseTools {
		toolSet[t] = true
	}

	schemas := make([]map[string]any, 0, len(toolSet))
	for name := range toolSet {
		t, ok := r.tools[name]
		if !ok || !t.ReadOnly() {
			continue
		}
		schemas = append(schemas, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  t.ParametersSchema(),
			},
		})
	}
	return schemas
}

// getPhaseBaseTools 返回所有阶段都需要的核心只读工具
func (r *ToolRegistry) getPhaseBaseTools() []string {
	return []string{
		"get_actor",
		"get_pc",
		"list_actors",
		"get_phase",
	}
}

// getPhaseSpecificTools 返回特定阶段额外需要的只读工具
func (r *ToolRegistry) getPhaseSpecificTools(phase string) []string {
	switch phase {
	case "character_creation":
		return []string{
			"list_races", "get_race",
			"list_classes", "get_class",
			"list_backgrounds", "get_background",
			"list_feats_data", "get_feat_data",
		}
	case "exploration":
		return []string{
			"get_current_scene", "get_scene", "list_scenes",
			"get_scene_actors", "get_scene_items",
			"get_inventory", "get_equipment",
			"get_passive_perception",
			"get_npc_attitude",
			"get_quest", "list_quests", "get_actor_quests",
			"get_magic_item_bonus",
			"calculate_breath_holding", "calculate_mount_speed",
			"get_crafting_recipes",
			"list_spells", "get_spell",
			"list_weapons", "list_armors", "list_magic_items",
			"list_monsters", "get_monster",
		}
	case "combat":
		return []string{
			"get_current_combat", "get_current_turn",
			"get_actor", "get_pc", "list_actors",
			"get_inventory", "get_equipment",
			"get_spell_slots",
			"get_passive_perception",
			"list_spells", "get_spell",
		}
	case "rest":
		return []string{
			"get_actor", "get_pc",
			"get_inventory", "get_equipment",
			"get_spell_slots",
			"get_quest", "list_quests", "get_actor_quests",
		}
	default:
		// 未知阶段，返回 exploration 作为默认
		return r.getPhaseSpecificTools("exploration")
	}
}

// ExecuteTools 执行多个Tool调用
func (r *ToolRegistry) ExecuteTools(ctx context.Context, calls []llm.ToolCall) []llm.ToolResult {
	return r.executeTools(ctx, "", calls)
}

// ExecuteToolsForAgent 按 Agent 权限执行多个 Tool 调用。
func (r *ToolRegistry) ExecuteToolsForAgent(ctx context.Context, agentName string, calls []llm.ToolCall) []llm.ToolResult {
	return r.executeTools(ctx, agentName, calls)
}

func (r *ToolRegistry) executeTools(ctx context.Context, agentName string, calls []llm.ToolCall) []llm.ToolResult {
	log := r.getLogger()

	log.Debug("[ToolRegistry] ExecuteTools started",
		zap.Int("callCount", len(calls)),
		zap.String("agentName", agentName),
	)

	results := make([]llm.ToolResult, 0, len(calls))

	for i, call := range calls {
		argsJSON, _ := json.Marshal(call.Arguments)
		log.Debug("[ToolRegistry] Executing tool",
			zap.Int("index", i),
			zap.String("toolName", call.Name),
			zap.String("toolCallID", call.ID),
			zap.String("arguments", truncateForLog(string(argsJSON), 200)),
		)

		if agentName != "" && !r.IsToolAllowedForAgent(agentName, call.Name) {
			result := llm.ToolResult{
				ToolCallID: call.ID,
				Content:    "Error: tool not allowed for agent: " + agentName + " cannot call " + call.Name,
				IsError:    true,
			}
			results = append(results, result)
			log.Warn("[ToolRegistry] Tool call rejected by agent permission",
				zap.String("agentName", agentName),
				zap.String("toolName", call.Name),
				zap.String("toolCallID", call.ID),
			)
			continue
		}

		result := r.executeTool(ctx, call)
		results = append(results, result)

		log.Debug("[ToolRegistry] Tool executed",
			zap.String("toolName", call.Name),
			zap.String("toolCallID", call.ID),
			zap.Bool("isError", result.IsError),
			zap.String("content", truncateForLog(result.Content, 200)),
		)
	}

	log.Debug("[ToolRegistry] ExecuteTools completed",
		zap.Int("totalCalls", len(calls)),
		zap.Int("errorCount", countErrors(results)),
	)

	return results
}

// truncateForLog 截断日志字符串
func truncateForLog(s string, maxLen int) string {
	if s == "" {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// countErrors 统计错误数量
func countErrors(results []llm.ToolResult) int {
	count := 0
	for _, r := range results {
		if r.IsError {
			count++
		}
	}
	return count
}

// executeTool 执行单个Tool调用
func (r *ToolRegistry) executeTool(ctx context.Context, call llm.ToolCall) llm.ToolResult {
	tool, ok := r.Get(call.Name)
	if !ok {
		return llm.ToolResult{
			ToolCallID: call.ID,
			Content:    "Error: tool not found: " + call.Name,
			IsError:    true,
		}
	}

	toolResult, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		return llm.ToolResult{
			ToolCallID: call.ID,
			Content:    "Error: " + err.Error(),
			IsError:    true,
		}
	}

	if !toolResult.Success {
		content := "Error: " + toolResult.Error
		if toolResult.Data != nil {
			dataJSON, _ := json.Marshal(toolResult.Data)
			if len(dataJSON) > 0 && string(dataJSON) != "null" {
				content += "\n" + string(dataJSON)
			}
		}
		return llm.ToolResult{
			ToolCallID: call.ID,
			Content:    content,
			IsError:    true,
		}
	}

	// 格式化结果为字符串
	// 优先使用 Message，但同时包含 Data（如果有）
	var content string
	if toolResult.Message != "" && toolResult.Data != nil {
		// 同时包含消息和数据
		dataJSON, _ := json.Marshal(toolResult.Data)
		content = toolResult.Message + "\n" + string(dataJSON)
	} else if toolResult.Message != "" {
		content = toolResult.Message
	} else if toolResult.Data != nil {
		data, _ := json.Marshal(toolResult.Data)
		content = string(data)
	}

	return llm.ToolResult{
		ToolCallID: call.ID,
		Content:    content,
		IsError:    false,
	}
}
