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

// ExecuteTools 执行多个Tool调用
func (r *ToolRegistry) ExecuteTools(ctx context.Context, calls []llm.ToolCall) []llm.ToolResult {
	log := r.getLogger()

	log.Debug("[ToolRegistry] ExecuteTools started",
		zap.Int("callCount", len(calls)),
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
		return llm.ToolResult{
			ToolCallID: call.ID,
			Content:    "Error: " + toolResult.Error,
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
