package tool

import (
	"context"
	"encoding/json"

	"github.com/zwh8800/cdndv2/game_engine/llm"
)

// ToolRegistry Tool注册中心
type ToolRegistry struct {
	tools    map[string]Tool
	byAgent  map[string][]string // agent -> tool names
	category map[string][]string // category -> tool names
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
	results := make([]llm.ToolResult, 0, len(calls))

	for _, call := range calls {
		result := r.executeTool(ctx, call)
		results = append(results, result)
	}

	return results
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
	content := toolResult.Message
	if content == "" && toolResult.Data != nil {
		data, _ := json.Marshal(toolResult.Data)
		content = string(data)
	}

	return llm.ToolResult{
		ToolCallID: call.ID,
		Content:    content,
		IsError:    false,
	}
}
