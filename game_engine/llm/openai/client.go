package openai

import (
	"context"
	"errors"
	"fmt"

	"github.com/zwh8800/cdndv2/game_engine/llm"
)

var (
	ErrNotImplemented = errors.New("openai client not fully implemented - use mock client for testing")
	ErrMissingAPIKey  = errors.New("missing OpenAI API key")
)

// OpenAIClient OpenAI API客户端实现 (占位实现)
type OpenAIClient struct {
	config OpenAIConfig
}

// NewOpenAIClient 创建OpenAI客户端
func NewOpenAIClient(config OpenAIConfig) (*OpenAIClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &OpenAIClient{
		config: config,
	}, nil
}

// Complete 执行非流式完成请求
func (c *OpenAIClient) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	// TODO: 实现完整的OpenAI API调用
	// 需要适配 openai-go SDK v1.12.0 的新 API
	return nil, ErrNotImplemented
}

// Stream 执行流式完成请求
func (c *OpenAIClient) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	// TODO: 实现完整的OpenAI Stream API调用
	return nil, ErrNotImplemented
}

// GetConfig 获取配置
func (c *OpenAIClient) GetConfig() OpenAIConfig {
	return c.config
}

// MockClient 模拟LLM客户端（用于测试）
type MockClient struct {
	Responses     []*llm.CompletionResponse
	ResponseIndex int
}

// NewMockClient 创建模拟客户端
func NewMockClient(responses []*llm.CompletionResponse) *MockClient {
	return &MockClient{
		Responses: responses,
	}
}

// Complete 模拟完成请求
func (m *MockClient) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.ResponseIndex < len(m.Responses) {
		resp := m.Responses[m.ResponseIndex]
		m.ResponseIndex++
		return resp, nil
	}
	return &llm.CompletionResponse{
		Content:      "Mock response",
		FinishReason: llm.FinishReasonStop,
	}, nil
}

// Stream 模拟流式请求
func (m *MockClient) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- llm.StreamChunk{Delta: "Mock stream response", Done: false}
		ch <- llm.StreamChunk{Done: true}
	}()
	return ch, nil
}

// AddResponse 添加模拟响应
func (m *MockClient) AddResponse(resp *llm.CompletionResponse) {
	m.Responses = append(m.Responses, resp)
}

// String 返回客户端信息
func (c *OpenAIClient) String() string {
	return fmt.Sprintf("OpenAIClient(model=%s)", c.config.Model)
}
