package llm

import (
	"encoding/json"
	"fmt"
)

// ParseToolCalls 从LLM原始响应解析Tool调用
// 通常用于处理原始JSON响应中的tool_calls字段
func ParseToolCalls(raw any) ([]ToolCall, error) {
	if raw == nil {
		return nil, nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool calls: %w", err)
	}

	var calls []ToolCall
	if err := json.Unmarshal(data, &calls); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool calls: %w", err)
	}

	return calls, nil
}

// FormatToolResult 格式化Tool结果为LLM可理解的字符串
func FormatToolResult(result *ToolResult) string {
	if result == nil {
		return "Tool result is nil"
	}

	if result.IsError {
		return fmt.Sprintf("Error: %s", result.Content)
	}

	return result.Content
}

// ExtractContent 从CompletionResponse提取纯文本内容
func ExtractContent(response *CompletionResponse) string {
	if response == nil {
		return ""
	}
	return response.Content
}

// ToolResultToString 将ToolResult转换为字符串表示
func ToolResultToString(result *ToolResult) string {
	if result == nil {
		return "null"
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("{error: %v}", err)
	}

	return string(data)
}

// MessagesToStrings 将消息列表转换为可读字符串（用于调试）
func MessagesToStrings(messages []Message) string {
	result := ""
	for _, msg := range messages {
		result += fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content)
	}
	return result
}
