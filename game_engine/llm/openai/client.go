package openai

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/zwh8800/cdndv2/game_engine/llm"
)

var (
	ErrMissingAPIKey = fmt.Errorf("missing OpenAI API key")
)

// OpenAIClient OpenAI API客户端实现
type OpenAIClient struct {
	client openai.Client
	config OpenAIConfig
}

// NewOpenAIClient 创建OpenAI客户端
func NewOpenAIClient(config OpenAIConfig) (*OpenAIClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 构建客户端选项
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}

	// 如果配置了自定义 BaseURL
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	return &OpenAIClient{
		client: openai.NewClient(opts...),
		config: config,
	}, nil
}

// Complete 执行非流式完成请求
func (c *OpenAIClient) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	// 构建消息
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, c.convertMessage(msg))
	}

	// 构建工具定义
	var tools []openai.ChatCompletionToolParam
	if len(req.Tools) > 0 {
		tools = make([]openai.ChatCompletionToolParam, 0, len(req.Tools))
		for _, toolDef := range req.Tools {
			if tool, err := c.convertTool(toolDef); err == nil {
				tools = append(tools, tool)
			}
		}
	}

	// 构建请求参数
	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    openai.ChatModel(c.config.Model),
	}

	// 设置可选参数
	if c.config.Temperature > 0 {
		params.Temperature = openai.Float(c.config.Temperature)
	}
	if c.config.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(c.config.MaxTokens))
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	// 调用 OpenAI API
	completion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion failed: %w", err)
	}

	// 解析响应
	return c.parseCompletionResponse(completion)
}

// Stream 执行流式完成请求
func (c *OpenAIClient) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	// 构建消息
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, c.convertMessage(msg))
	}

	// 构建工具定义
	var tools []openai.ChatCompletionToolParam
	if len(req.Tools) > 0 {
		tools = make([]openai.ChatCompletionToolParam, 0, len(req.Tools))
		for _, toolDef := range req.Tools {
			if tool, err := c.convertTool(toolDef); err == nil {
				tools = append(tools, tool)
			}
		}
	}

	// 构建请求参数
	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    openai.ChatModel(c.config.Model),
	}

	if c.config.Temperature > 0 {
		params.Temperature = openai.Float(c.config.Temperature)
	}
	if c.config.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(c.config.MaxTokens))
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	// 创建流式通道
	ch := make(chan llm.StreamChunk, 32)

	// 启动流式请求
	go func() {
		defer close(ch)

		stream := c.client.Chat.Completions.NewStreaming(ctx, params)
		defer stream.Close()

		for stream.Next() {
			chunk := stream.Current()

			// 处理每个 choice
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					ch <- llm.StreamChunk{
						Delta: choice.Delta.Content,
						Done:  false,
					}
				}

				// 检查是否完成
				if choice.FinishReason != "" {
					ch <- llm.StreamChunk{
						Delta: "",
						Done:  true,
					}
				}
			}
		}

		// 检查流式错误
		if err := stream.Err(); err != nil {
			ch <- llm.StreamChunk{
				Delta: fmt.Sprintf("stream error: %v", err),
				Done:  true,
			}
		}
	}()

	return ch, nil
}

// convertMessage 将内部消息转换为 OpenAI 消息格式
func (c *OpenAIClient) convertMessage(msg llm.Message) openai.ChatCompletionMessageParamUnion {
	switch msg.Role {
	case llm.RoleSystem:
		return openai.SystemMessage(msg.Content)
	case llm.RoleUser:
		return openai.UserMessage(msg.Content)
	case llm.RoleAssistant:
		if len(msg.ToolCalls) > 0 {
			// 转换 tool calls
			toolCalls := make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				argsBytes, _ := json.Marshal(tc.Arguments)
				toolCalls[i] = openai.ChatCompletionMessageToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      tc.Name,
						Arguments: string(argsBytes),
					},
				}
			}
			return openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					},
					ToolCalls: toolCalls,
				},
			}
		}
		return openai.AssistantMessage(msg.Content)
	case llm.RoleTool:
		return openai.ToolMessage(msg.Content, msg.Name)
	default:
		return openai.UserMessage(msg.Content)
	}
}

// convertTool 将工具定义转换为 OpenAI 工具格式
func (c *OpenAIClient) convertTool(toolDef map[string]any) (openai.ChatCompletionToolParam, error) {
	tool := openai.ChatCompletionToolParam{}

	// 提取 function 定义
	if functionDef, ok := toolDef["function"].(map[string]any); ok {
		name, _ := functionDef["name"].(string)
		description, _ := functionDef["description"].(string)
		parameters, _ := functionDef["parameters"].(map[string]any)

		// 转换 parameters 为 JSON Schema 格式
		var params openai.FunctionParameters
		if parameters != nil {
			// 先将 map 转为 JSON，再解析为 FunctionParameters
			jsonBytes, err := json.Marshal(parameters)
			if err != nil {
				return tool, fmt.Errorf("failed to marshal parameters: %w", err)
			}
			if err := json.Unmarshal(jsonBytes, &params); err != nil {
				return tool, fmt.Errorf("failed to unmarshal parameters: %w", err)
			}
		}

		tool.Function = openai.FunctionDefinitionParam{
			Name:        name,
			Description: openai.String(description),
			Parameters:  params,
		}
	}

	return tool, nil
}

// parseCompletionResponse 解析 OpenAI 完成响应
func (c *OpenAIClient) parseCompletionResponse(completion *openai.ChatCompletion) (*llm.CompletionResponse, error) {
	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no choices in completion response")
	}

	choice := completion.Choices[0]
	response := &llm.CompletionResponse{
		Content: choice.Message.Content,
	}

	// 解析 tool calls
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]llm.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			var arguments map[string]any
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					return nil, fmt.Errorf("failed to parse tool call arguments: %w", err)
				}
			}

			response.ToolCalls[i] = llm.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: arguments,
			}
		}
	}

	// 解析 finish reason
	response.FinishReason = convertFinishReason(choice.FinishReason)

	// 解析 token 使用信息（Usage 是值类型，检查 PromptTokens 是否为 0）
	if completion.Usage.PromptTokens > 0 || completion.Usage.CompletionTokens > 0 {
		response.Usage = llm.Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		}
	}

	return response, nil
}

// GetConfig 获取配置
func (c *OpenAIClient) GetConfig() OpenAIConfig {
	return c.config
}

// String 返回客户端信息
func (c *OpenAIClient) String() string {
	return fmt.Sprintf("OpenAIClient(model=%s)", c.config.Model)
}

// convertFinishReason 辅助函数：转换字符串为 FinishReason
func convertFinishReason(reason string) llm.FinishReason {
	switch reason {
	case "stop":
		return llm.FinishReasonStop
	case "tool_calls":
		return llm.FinishReasonToolCalls
	case "length":
		return llm.FinishReasonLength
	case "content_filter":
		return llm.FinishReasonError
	default:
		return llm.FinishReasonStop
	}
}
