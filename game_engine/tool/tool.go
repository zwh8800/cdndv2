package tool

import (
	"context"
)

// Tool 基础接口
type Tool interface {
	// Name 返回Tool名称
	Name() string

	// Description 返回Tool描述
	Description() string

	// ParametersSchema 返回参数JSON Schema
	ParametersSchema() map[string]any

	// Execute 执行Tool
	Execute(ctx context.Context, params map[string]any) (*ToolResult, error)
}

// ToolResult Tool执行结果
type ToolResult struct {
	Success  bool           `json:"success"`
	Data     any            `json:"data"`
	Message  string         `json:"message"`
	Error    string         `json:"error,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// BaseTool 基础Tool实现
type BaseTool struct {
	name        string
	description string
	schema      map[string]any
}

func (t *BaseTool) Name() string {
	return t.name
}

func (t *BaseTool) Description() string {
	return t.description
}

func (t *BaseTool) ParametersSchema() map[string]any {
	return t.schema
}

// EngineTool 引擎Tool基类
type EngineTool struct {
	BaseTool
	engine any // *engine.Engine，使用any避免循环依赖
}

func NewEngineTool(name, description string, schema map[string]any, engine any) *EngineTool {
	return &EngineTool{
		BaseTool: BaseTool{
			name:        name,
			description: description,
			schema:      schema,
		},
		engine: engine,
	}
}

func (t *EngineTool) Engine() any {
	return t.engine
}
