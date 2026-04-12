package llm

import (
	"context"
)

// Usage Token使用信息
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// FinishReason 完成原因
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonToolCalls FinishReason = "tool_calls"
	FinishReasonLength    FinishReason = "length"
	FinishReasonError     FinishReason = "error"
)

// CompletionRequest LLM完成请求
type CompletionRequest struct {
	Messages    []Message        `json:"messages"`
	Tools       []map[string]any `json:"tools,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

// CompletionResponse LLM完成响应
type CompletionResponse struct {
	Content      string       `json:"content"`
	ToolCalls    []ToolCall   `json:"tool_calls,omitempty"`
	Usage        Usage        `json:"usage"`
	FinishReason FinishReason `json:"finish_reason"`
}

// StreamChunk 流式输出块
type StreamChunk struct {
	Delta    string    `json:"delta"`
	Done     bool      `json:"done"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
}

// LLMClient LLM客户端接口
type LLMClient interface {
	// Complete 执行非流式完成请求
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Stream 执行流式完成请求
	Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)
}
