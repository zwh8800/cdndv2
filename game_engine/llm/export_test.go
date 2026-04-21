package llm

import "context"

// export_test.go 暴露内部方法供外部测试包 llm_test 使用
// 这是 Go 标准的测试桥模式

// ExportIdentifyRounds 暴露 identifyRounds
func (c *ContextCompressor) ExportIdentifyRounds(msgs []Message) []MessageRound {
	rounds := c.identifyRounds(msgs)
	result := make([]MessageRound, len(rounds))
	for i, r := range rounds {
		result[i] = MessageRound{Messages: r.messages}
	}
	return result
}

// MessageRound 暴露 messageRound 结构
type MessageRound struct {
	Messages []Message
}

// ExportSummarizeToolSequence 暴露 summarizeToolSequence
func (c *ContextCompressor) ExportSummarizeToolSequence(msgs []Message) string {
	return c.summarizeToolSequence(msgs)
}

// ExportSummarizeSegment 暴露 summarizeSegment
func (c *ContextCompressor) ExportSummarizeSegment(msgs []Message, segType segmentType) string {
	return c.summarizeSegment(messageSegment{messages: msgs, segType: segType})
}

// ExportSummarizeWithLLM 暴露 summarizeWithLLM
func (c *ContextCompressor) ExportSummarizeWithLLM(ctx context.Context, msgs []Message) (string, error) {
	return c.summarizeWithLLM(ctx, msgs)
}

// ExportBuildSummarizationPrompt 暴露 buildSummarizationPrompt
func (c *ContextCompressor) ExportBuildSummarizationPrompt(msgs []Message) string {
	return c.buildSummarizationPrompt(msgs)
}

// ExportPruneOldMessages 暴露 pruneOldMessages
func (c *ContextCompressor) ExportPruneOldMessages(msgs []Message) []Message {
	return c.pruneOldMessages(msgs)
}

// ExportCompressOldMessagesHeuristic 暴露 compressOldMessagesHeuristic
func (c *ContextCompressor) ExportCompressOldMessagesHeuristic(msgs []Message) []Message {
	return c.compressOldMessagesHeuristic(msgs)
}

// ExportSegmentMessages 暴露 segmentMessages
func (c *ContextCompressor) ExportSegmentMessages(msgs []Message) int {
	return len(c.segmentMessages(msgs))
}

// ExportIsQueryTool 暴露 isQueryTool 方法（现在需要 ContextCompressor 实例）
// 向后兼容：创建一个无 registry 的压缩器，回退到前缀匹配行为
func ExportIsQueryTool(name string) bool {
	c := &ContextCompressor{}
	return c.isQueryTool(name)
}

// ExportContainsEntityID 暴露 containsEntityID
func ExportContainsEntityID(content string) bool {
	return containsEntityID(content)
}

// ExportEstimateStringTokens 暴露 estimateStringTokens
func ExportEstimateStringTokens(s string) int {
	return estimateStringTokens(s)
}

// 暴露 segment type 常量
var (
	ExportSegTypeUserInput = segTypeUserInput
	ExportSegTypeAssistant = segTypeAssistant
)

// GetCalibrationRatio 获取校准系数
func (c *ContextCompressor) GetCalibrationRatio() float64 {
	return c.calibrationRatio
}

// SetCalibrationRatio 设置校准系数
func (c *ContextCompressor) SetCalibrationRatio(ratio float64) {
	c.calibrationRatio = ratio
}

// GetLLMClient 获取 LLM 客户端
func (c *ContextCompressor) GetLLMClient() LLMClient {
	return c.llmClient
}
