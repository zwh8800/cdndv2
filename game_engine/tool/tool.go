package tool

import (
	"context"
	"fmt"
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

// ========== 参数安全提取辅助函数 ==========

// ParamError 参数缺失或类型错误
type ParamError struct {
	Key     string
	Message string
}

func (e *ParamError) Error() string {
	return fmt.Sprintf("parameter %s: %s", e.Key, e.Message)
}

// RequireString 安全提取必填字符串参数
func RequireString(params map[string]any, key string) (string, error) {
	v, ok := params[key]
	if !ok {
		return "", &ParamError{Key: key, Message: "missing required parameter"}
	}
	s, ok := v.(string)
	if !ok {
		return "", &ParamError{Key: key, Message: fmt.Sprintf("expected string, got %T", v)}
	}
	return s, nil
}

// RequireFloat 安全提取必填float64参数
func RequireFloat(params map[string]any, key string) (float64, error) {
	v, ok := params[key]
	if !ok {
		return 0, &ParamError{Key: key, Message: "missing required parameter"}
	}
	f, ok := v.(float64)
	if !ok {
		return 0, &ParamError{Key: key, Message: fmt.Sprintf("expected number, got %T", v)}
	}
	return f, nil
}

// RequireInt 安全提取必填int参数（从float64转换）
func RequireInt(params map[string]any, key string) (int, error) {
	f, err := RequireFloat(params, key)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}

// OptionalString 安全提取可选字符串参数
func OptionalString(params map[string]any, key string, defaultVal string) string {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}

// OptionalInt 安全提取可选int参数
func OptionalInt(params map[string]any, key string, defaultVal int) int {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	f, ok := v.(float64)
	if !ok {
		return defaultVal
	}
	return int(f)
}

// OptionalBool 安全提取可选bool参数
func OptionalBool(params map[string]any, key string, defaultVal bool) bool {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	b, ok := v.(bool)
	if !ok {
		return defaultVal
	}
	return b
}

// OptionalFloat 安全提取可选float64参数
func OptionalFloat(params map[string]any, key string, defaultVal float64) float64 {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	f, ok := v.(float64)
	if !ok {
		return defaultVal
	}
	return f
}

// RequireStringArray 安全提取必填字符串数组参数
func RequireStringArray(params map[string]any, key string) ([]string, error) {
	v, ok := params[key]
	if !ok {
		return nil, &ParamError{Key: key, Message: "missing required parameter"}
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, &ParamError{Key: key, Message: fmt.Sprintf("expected array, got %T", v)}
	}
	result := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, &ParamError{Key: key, Message: fmt.Sprintf("expected string at index %d, got %T", i, item)}
		}
		result[i] = s
	}
	return result, nil
}

// OptionalStringArray 安全提取可选字符串数组参数
func OptionalStringArray(params map[string]any, key string) []string {
	v, ok := params[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// RequireMap 安全提取必填map参数
func RequireMap(params map[string]any, key string) (map[string]any, error) {
	v, ok := params[key]
	if !ok {
		return nil, &ParamError{Key: key, Message: "missing required parameter"}
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, &ParamError{Key: key, Message: fmt.Sprintf("expected object, got %T", v)}
	}
	return m, nil
}
