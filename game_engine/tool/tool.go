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

	// ReadOnly 返回是否为只读工具（不修改游戏状态）
	ReadOnly() bool
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
	readOnly    bool
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

// ReadOnly 返回是否为只读工具（默认 false）
func (t *BaseTool) ReadOnly() bool {
	return t.readOnly
}

// EngineTool 引擎Tool基类
type EngineTool struct {
	BaseTool
	engine any // *engine.Engine，使用any避免循环依赖
}

func NewEngineTool(name, description string, schema map[string]any, engine any, readOnly bool) *EngineTool {
	return &EngineTool{
		BaseTool: BaseTool{
			name:        name,
			description: description,
			schema:      schema,
			readOnly:    readOnly,
		},
		engine: engine,
	}
}

func (t *EngineTool) Engine() any {
	return t.engine
}
