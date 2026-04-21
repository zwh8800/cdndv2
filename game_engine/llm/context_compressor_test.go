package llm_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/llm/openai"
)

// --- 真实 LLMClient 辅助函数 ---

func getOpenAIConfigFromEnv() openai.OpenAIConfig {
	config := openai.DefaultOpenAIConfig()
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		config.Model = model
	}
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}
	return config
}

func newRealLLMClient(t *testing.T) llm.LLMClient {
	t.Helper()
	config := getOpenAIConfigFromEnv()
	if config.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}
	t.Logf("[LLM] 使用模型: %s, BaseURL: %s", config.Model, config.BaseURL)
	client, err := openai.NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}
	return client
}

// --- slowMockLLMClient 仅用于异步并发测试（不涉及 LLM 摘要质量） ---

type slowMockLLMClient struct {
	callCount atomic.Int32
	delay     time.Duration
}

func (m *slowMockLLMClient) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	m.callCount.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return &llm.CompletionResponse{Content: "mock summary for concurrency test"}, nil
}

func (m *slowMockLLMClient) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	return nil, fmt.Errorf("not implemented")
}

// --- 辅助函数 ---

// roundMsgCounts 返回每轮的消息数，用于日志
func roundMsgCounts(rounds []llm.MessageRound) []int {
	counts := make([]int, len(rounds))
	for i, r := range rounds {
		counts[i] = len(r.Messages)
	}
	return counts
}

func makeHistory(rounds int) []llm.Message {
	var msgs []llm.Message
	for i := 0; i < rounds; i++ {
		msgs = append(msgs, llm.NewUserMessage(fmt.Sprintf("玩家输入第%d轮", i+1)))
		msgs = append(msgs, llm.NewAssistantMessage(fmt.Sprintf("DM回复第%d轮", i+1), nil))
	}
	return msgs
}

func makeToolCallHistory() []llm.Message {
	return []llm.Message{
		llm.NewUserMessage("查看角色状态"),
		llm.NewAssistantMessage("", []llm.ToolCall{
			{ID: "call_1", Name: "get_character", Arguments: map[string]any{"actor_id": "01H123"}},
		}),
		{Role: llm.RoleTool, Content: `{"name":"勇者","hp":50,"actor_id":"01H123"}`, Name: "call_1"},
		llm.NewAssistantMessage("你的角色勇者当前HP为50", nil),
		llm.NewUserMessage("攻击哥布林"),
		llm.NewAssistantMessage("", []llm.ToolCall{
			{ID: "call_2", Name: "attack", Arguments: map[string]any{"target_id": "01H456"}},
		}),
		{Role: llm.RoleTool, Content: `{"damage":15,"hit":true,"actor_id":"01H123","target_id":"01H456"}`, Name: "call_2"},
		llm.NewAssistantMessage("你成功攻击了哥布林，造成15点伤害", nil),
	}
}

func makeDnDHistory() []llm.Message {
	return []llm.Message{
		llm.NewUserMessage("我是一名人类战士Aldric，力量16，HP 30"),
		llm.NewAssistantMessage("欢迎Aldric！你站在一个古老村庄的广场上。", nil),
		llm.NewUserMessage("我向酒馆走去"),
		llm.NewAssistantMessage("", []llm.ToolCall{
			{ID: "c1", Name: "get_scene", Arguments: map[string]any{"scene_id": "01HSCENE1"}},
		}),
		{Role: llm.RoleTool, Content: `{"scene_id":"01HSCENE1","name":"破旧酒馆","description":"一间昏暗的酒馆"}`, Name: "c1"},
		llm.NewAssistantMessage("你走进了破旧酒馆，里面坐着几个冒险者和一位酒保。", nil),
		llm.NewUserMessage("我向酒保询问最近的冒险任务"),
		llm.NewAssistantMessage("酒保告诉你北方森林里有一群哥布林在骚扰旅人，悬赏50金币。", nil),
		llm.NewUserMessage("我接受任务，前往北方森林"),
		llm.NewAssistantMessage("", []llm.ToolCall{
			{ID: "c2", Name: "move_to", Arguments: map[string]any{"destination": "north_forest", "actor_id": "01HACTOR1"}},
		}),
		{Role: llm.RoleTool, Content: `{"success":true,"actor_id":"01HACTOR1","new_scene":"01HSCENE2"}`, Name: "c2"},
		llm.NewAssistantMessage("你来到了北方森林的边缘。树木茂密，隐约可以听到远处的响动。", nil),
		llm.NewUserMessage("我小心地向声音的方向前进"),
		llm.NewAssistantMessage("你发现了三只哥布林在路旁设伏！战斗开始！", nil),
		llm.NewAssistantMessage("", []llm.ToolCall{
			{ID: "c3", Name: "attack", Arguments: map[string]any{"actor_id": "01HACTOR1", "target_id": "01HGOBLIN1"}},
		}),
		{Role: llm.RoleTool, Content: `{"hit":true,"damage":12,"actor_id":"01HACTOR1","target_id":"01HGOBLIN1","target_hp_remaining":3}`, Name: "c3"},
		llm.NewAssistantMessage("你挥剑斩向第一只哥布林，造成12点伤害！它还剩3点HP。", nil),
	}
}

// =====================================================
// 1. 基础功能测试
// =====================================================

func TestDefaultContextCompressor(t *testing.T) {
	t.Run("nil LLMClient", func(t *testing.T) {
		t.Log("[初始化] 测试 nil LLMClient 默认配置")
		c := llm.DefaultContextCompressor(nil)
		t.Logf("[初始化] ContextWindowSize=%d, CompressThreshold=%.2f, RecentKeepRounds=%d, CalibrationRatio=%.2f",
			c.ContextWindowSize, c.CompressThreshold, c.RecentKeepRounds, c.GetCalibrationRatio())
		if c.ContextWindowSize != 128000 {
			t.Errorf("expected ContextWindowSize=128000, got %d", c.ContextWindowSize)
		}
		if c.CompressThreshold != 0.75 {
			t.Errorf("expected CompressThreshold=0.75, got %f", c.CompressThreshold)
		}
		if c.RecentKeepRounds != 3 {
			t.Errorf("expected RecentKeepRounds=3, got %d", c.RecentKeepRounds)
		}
		if c.GetCalibrationRatio() != 1.3 {
			t.Errorf("expected calibrationRatio=1.3, got %f", c.GetCalibrationRatio())
		}
		if c.GetLLMClient() != nil {
			t.Error("expected llmClient to be nil")
		}
		t.Log("[初始化] nil LLMClient 默认配置验证通过")
	})

	t.Run("with real LLMClient", func(t *testing.T) {
		t.Log("[初始化] 测试使用真实 LLMClient 初始化")
		client := newRealLLMClient(t)
		c := llm.DefaultContextCompressor(client)
		if c.GetLLMClient() == nil {
			t.Error("expected llmClient to be set")
		}
		t.Log("[初始化] 真实 LLMClient 初始化验证通过")
	})
}

func TestEstimateTokens(t *testing.T) {
	c := llm.DefaultContextCompressor(nil)

	t.Run("empty messages", func(t *testing.T) {
		t.Log("[Token估算] 测试空消息")
		tokens := c.EstimateTokens(nil)
		t.Logf("[Token估算] nil -> %d tokens", tokens)
		if tokens != 0 {
			t.Errorf("expected 0 tokens for nil messages, got %d", tokens)
		}
		tokens = c.EstimateTokens([]llm.Message{})
		t.Logf("[Token估算] [] -> %d tokens", tokens)
		if tokens != 0 {
			t.Errorf("expected 0 tokens for empty messages, got %d", tokens)
		}
	})

	t.Run("single message", func(t *testing.T) {
		msgs := []llm.Message{llm.NewUserMessage("hello world")}
		tokens := c.EstimateTokens(msgs)
		t.Logf("[Token估算] 'hello world' -> %d tokens", tokens)
		if tokens <= 0 {
			t.Errorf("expected positive tokens, got %d", tokens)
		}
	})

	t.Run("chinese text higher than english per char", func(t *testing.T) {
		chMsg := []llm.Message{llm.NewUserMessage("你好世界测试")}
		enMsg := []llm.Message{llm.NewUserMessage("abcdef")}
		chTokens := c.EstimateTokens(chMsg)
		enTokens := c.EstimateTokens(enMsg)
		t.Logf("[Token估算] 中文6字=%d tokens, 英文6字=%d tokens", chTokens, enTokens)
		if chTokens <= enTokens {
			t.Errorf("expected chinese tokens(%d) > english tokens(%d)", chTokens, enTokens)
		}
	})

	t.Run("message with tool calls", func(t *testing.T) {
		plain := []llm.Message{llm.NewAssistantMessage("response", nil)}
		withTools := []llm.Message{llm.NewAssistantMessage("response", []llm.ToolCall{
			{ID: "call_1", Name: "get_character", Arguments: map[string]any{"id": "123"}},
		})}
		plainTokens := c.EstimateTokens(plain)
		toolTokens := c.EstimateTokens(withTools)
		t.Logf("[Token估算] 纯文本=%d tokens, 含工具调用=%d tokens", plainTokens, toolTokens)
		if toolTokens <= plainTokens {
			t.Errorf("expected tool message tokens(%d) > plain(%d)", toolTokens, plainTokens)
		}
	})

	t.Run("calibration ratio applied", func(t *testing.T) {
		c1 := llm.DefaultContextCompressor(nil)
		c1.SetCalibrationRatio(1.0)
		c2 := llm.DefaultContextCompressor(nil)
		c2.SetCalibrationRatio(2.0)
		msgs := []llm.Message{llm.NewUserMessage("test message for ratio")}
		t1 := c1.EstimateTokens(msgs)
		t2 := c2.EstimateTokens(msgs)
		t.Logf("[Token估算] ratio=1.0 -> %d tokens, ratio=2.0 -> %d tokens", t1, t2)
		if t2 != t1*2 {
			t.Errorf("expected ratio 2.0 to double tokens: got %d vs %d", t2, t1)
		}
	})
}

func TestNeedsCompression(t *testing.T) {
	c := llm.DefaultContextCompressor(nil)
	c.ContextWindowSize = 100
	c.CompressThreshold = 0.75
	c.SetCalibrationRatio(1.0)

	t.Run("below threshold", func(t *testing.T) {
		msgs := []llm.Message{llm.NewUserMessage("hi")}
		result := c.NeedsCompression(msgs)
		t.Logf("[压缩阈值] 短消息 -> NeedsCompression=%v (窗口=%d, 阈值=%.0f%%)", result, c.ContextWindowSize, c.CompressThreshold*100)
		if result {
			t.Error("should not need compression for small messages")
		}
	})

	t.Run("above threshold", func(t *testing.T) {
		var msgs []llm.Message
		for i := 0; i < 50; i++ {
			msgs = append(msgs, llm.NewUserMessage(strings.Repeat("a very long message to inflate tokens ", 10)))
		}
		tokens := c.EstimateTokens(msgs)
		result := c.NeedsCompression(msgs)
		t.Logf("[压缩阈值] 50条长消息, 估算tokens=%d, 阈值=%d -> NeedsCompression=%v", tokens, int(float64(c.ContextWindowSize)*c.CompressThreshold), result)
		if !result {
			t.Error("should need compression for large messages")
		}
	})
}

func TestIdentifyRounds(t *testing.T) {
	c := llm.DefaultContextCompressor(nil)

	t.Run("empty messages", func(t *testing.T) {
		rounds := c.ExportIdentifyRounds(nil)
		t.Logf("[轮次识别] nil -> %d 轮", len(rounds))
		if len(rounds) != 0 {
			t.Errorf("expected 0 rounds, got %d", len(rounds))
		}
	})

	t.Run("single user message", func(t *testing.T) {
		msgs := []llm.Message{llm.NewUserMessage("hello")}
		rounds := c.ExportIdentifyRounds(msgs)
		t.Logf("[轮次识别] 1条user消息 -> %d 轮, 每轮消息数: %v", len(rounds), roundMsgCounts(rounds))
		if len(rounds) != 1 {
			t.Fatalf("expected 1 round, got %d", len(rounds))
		}
		if len(rounds[0].Messages) != 1 {
			t.Errorf("expected 1 message in round, got %d", len(rounds[0].Messages))
		}
	})

	t.Run("multiple rounds", func(t *testing.T) {
		msgs := makeHistory(3)
		rounds := c.ExportIdentifyRounds(msgs)
		t.Logf("[轮次识别] %d条消息(3轮对话) -> %d 轮, 每轮消息数: %v", len(msgs), len(rounds), roundMsgCounts(rounds))
		if len(rounds) != 3 {
			t.Fatalf("expected 3 rounds, got %d", len(rounds))
		}
		for i, r := range rounds {
			if len(r.Messages) != 2 {
				t.Errorf("round %d: expected 2 messages, got %d", i, len(r.Messages))
			}
		}
	})

	t.Run("leading system message forms separate round", func(t *testing.T) {
		msgs := []llm.Message{
			llm.NewSystemMessage("system prompt"),
			llm.NewUserMessage("hello"),
			llm.NewAssistantMessage("hi", nil),
			llm.NewUserMessage("bye"),
		}
		rounds := c.ExportIdentifyRounds(msgs)
		t.Logf("[轮次识别] system+user+assistant+user -> %d 轮, 每轮消息数: %v", len(rounds), roundMsgCounts(rounds))
		if len(rounds) != 3 {
			t.Fatalf("expected 3 rounds, got %d", len(rounds))
		}
		if len(rounds[0].Messages) != 1 {
			t.Errorf("round 0: expected 1 message (system), got %d", len(rounds[0].Messages))
		}
	})

	t.Run("round with tool calls", func(t *testing.T) {
		msgs := makeToolCallHistory()
		rounds := c.ExportIdentifyRounds(msgs)
		t.Logf("[轮次识别] 含工具调用的对话(%d条消息) -> %d 轮, 每轮消息数: %v", len(msgs), len(rounds), roundMsgCounts(rounds))
		if len(rounds) != 2 {
			t.Fatalf("expected 2 rounds, got %d", len(rounds))
		}
		if len(rounds[0].Messages) != 4 {
			t.Errorf("round 0: expected 4 messages, got %d", len(rounds[0].Messages))
		}
	})
}

// =====================================================
// 2. 压缩策略测试
// =====================================================

func TestCompressHistoryRoundBased(t *testing.T) {
	c := llm.DefaultContextCompressor(nil)
	c.RecentKeepRounds = 2

	t.Run("fewer rounds than keep count", func(t *testing.T) {
		msgs := makeHistory(2)
		t.Logf("[轮次保留] 输入 %d 条消息(2轮), RecentKeepRounds=%d", len(msgs), c.RecentKeepRounds)
		result := c.CompressHistory(context.Background(), msgs)
		t.Logf("[轮次保留] 结果: %d 条消息 (不应压缩)", len(result))
		if len(result) != len(msgs) {
			t.Error("should not compress when rounds <= keep count")
		}
	})

	t.Run("more rounds than keep count", func(t *testing.T) {
		msgs := makeHistory(5)
		t.Logf("[轮次保留] 输入 %d 条消息(5轮), RecentKeepRounds=%d", len(msgs), c.RecentKeepRounds)
		result := c.CompressHistory(context.Background(), msgs)
		t.Logf("[轮次保留] 结果: %d -> %d 条消息", len(msgs), len(result))
		if len(result) >= len(msgs) {
			t.Error("compressed result should be shorter than original")
		}
		lastFour := result[len(result)-4:]
		t.Logf("[轮次保留] 保留的最近消息: [%s] [%s]", lastFour[0].Content, lastFour[2].Content)
		if lastFour[0].Content != "玩家输入第4轮" {
			t.Errorf("expected recent round 4, got: %s", lastFour[0].Content)
		}
		if lastFour[2].Content != "玩家输入第5轮" {
			t.Errorf("expected recent round 5, got: %s", lastFour[2].Content)
		}
	})

	t.Run("compressed message is system role", func(t *testing.T) {
		msgs := makeHistory(5)
		result := c.CompressHistory(context.Background(), msgs)
		t.Logf("[轮次保留] 摘要消息 Role=%s, 内容前80字: %.80s", result[0].Role, result[0].Content)
		if result[0].Role != llm.RoleSystem {
			t.Errorf("expected first message to be system role, got %s", result[0].Role)
		}
		if !strings.Contains(result[0].Content, "历史上下文摘要") {
			t.Error("expected summary header in compressed message")
		}
	})
}

func TestSummarizeToolSequence(t *testing.T) {
	c := llm.DefaultContextCompressor(nil)

	t.Run("query tools aggressive compression", func(t *testing.T) {
		msgs := []llm.Message{
			llm.NewAssistantMessage("", []llm.ToolCall{
				{ID: "c1", Name: "get_character", Arguments: map[string]any{}},
				{ID: "c2", Name: "list_items", Arguments: map[string]any{}},
			}),
			{Role: llm.RoleTool, Content: "char data...", Name: "c1"},
			{Role: llm.RoleTool, Content: "item list...", Name: "c2"},
		}
		summary := c.ExportSummarizeToolSequence(msgs)
		t.Logf("[工具摘要] 查询工具(get_character+list_items) -> %q", summary)
		if !strings.Contains(summary, "查询") {
			t.Error("expected query label")
		}
		if !strings.Contains(summary, "get_character") {
			t.Error("expected tool name get_character")
		}
	})

	t.Run("action tools conservative compression", func(t *testing.T) {
		msgs := []llm.Message{
			llm.NewAssistantMessage("", []llm.ToolCall{
				{ID: "c1", Name: "attack", Arguments: map[string]any{}},
			}),
			{Role: llm.RoleTool, Content: `{"damage":15,"actor_id":"01H123"}`, Name: "c1"},
		}
		summary := c.ExportSummarizeToolSequence(msgs)
		t.Logf("[工具摘要] 动作工具(attack) -> %q", summary)
		if !strings.Contains(summary, "执行") {
			t.Error("expected action label")
		}
		if !strings.Contains(summary, "attack") {
			t.Error("expected tool name attack")
		}
	})

	t.Run("error status detected", func(t *testing.T) {
		msgs := []llm.Message{
			llm.NewAssistantMessage("", []llm.ToolCall{
				{ID: "c1", Name: "cast_spell", Arguments: map[string]any{}},
			}),
			{Role: llm.RoleTool, Content: "Error: spell slot exhausted", Name: "c1"},
		}
		summary := c.ExportSummarizeToolSequence(msgs)
		t.Logf("[工具摘要] 错误工具(cast_spell) -> %q", summary)
		if !strings.Contains(summary, "部分失败") {
			t.Error("expected failure status")
		}
	})

	t.Run("empty messages", func(t *testing.T) {
		summary := c.ExportSummarizeToolSequence(nil)
		t.Logf("[工具摘要] nil -> %q", summary)
		if summary != "" {
			t.Errorf("expected empty summary, got: %s", summary)
		}
	})
}

func TestSummarizeSegment(t *testing.T) {
	c := llm.DefaultContextCompressor(nil)

	t.Run("user input short", func(t *testing.T) {
		msgs := []llm.Message{llm.NewUserMessage("攻击哥布林")}
		summary := c.ExportSummarizeSegment(msgs, llm.ExportSegTypeUserInput)
		t.Logf("[段摘要] 短玩家输入 -> %q", summary)
		if !strings.Contains(summary, "玩家") {
			t.Error("expected player label")
		}
		if !strings.Contains(summary, "攻击哥布林") {
			t.Error("expected original content preserved")
		}
	})

	t.Run("user input long truncated", func(t *testing.T) {
		longInput := strings.Repeat("这是一段很长的玩家输入", 30)
		msgs := []llm.Message{llm.NewUserMessage(longInput)}
		summary := c.ExportSummarizeSegment(msgs, llm.ExportSegTypeUserInput)
		t.Logf("[段摘要] 长玩家输入(%d字) -> 摘要长度=%d, 末尾: ...%s", len(longInput), len(summary), summary[max(0, len(summary)-20):])
		if !strings.HasSuffix(summary, "...") {
			t.Error("expected truncation suffix")
		}
	})

	t.Run("assistant response", func(t *testing.T) {
		msgs := []llm.Message{llm.NewAssistantMessage("你进入了一个黑暗的洞穴", nil)}
		summary := c.ExportSummarizeSegment(msgs, llm.ExportSegTypeAssistant)
		t.Logf("[段摘要] assistant响应 -> %q", summary)
		if !strings.Contains(summary, "DM") {
			t.Error("expected DM label")
		}
	})

	t.Run("empty assistant response", func(t *testing.T) {
		msgs := []llm.Message{llm.NewAssistantMessage("", nil)}
		summary := c.ExportSummarizeSegment(msgs, llm.ExportSegTypeAssistant)
		t.Logf("[段摘要] 空assistant -> %q", summary)
		if summary != "" {
			t.Errorf("expected empty summary, got: %s", summary)
		}
	})
}

func TestIsQueryTool(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"get_character", true},
		{"list_items", true},
		{"query_status", true},
		{"attack", false},
		{"cast_spell", false},
		{"", false},
	}
	t.Logf("[查询工具] 共 %d 个测试用例", len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := llm.ExportIsQueryTool(tt.name)
			t.Logf("[查询工具] isQueryTool(%q) = %v (期望 %v)", tt.name, got, tt.expected)
			if got != tt.expected {
				t.Errorf("isQueryTool(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestContainsEntityID(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{`{"actor_id":"01H123"}`, true},
		{`{"scene_id":"01HABC"}`, true},
		{`{"damage":15,"hit":true}`, false},
		{`simple text`, false},
		{``, false},
	}
	t.Logf("[实体ID] 共 %d 个测试用例", len(tests))
	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			got := llm.ExportContainsEntityID(tt.content)
			t.Logf("[实体ID] containsEntityID(%q) = %v (期望 %v)", tt.content, got, tt.expected)
			if got != tt.expected {
				t.Errorf("containsEntityID(%q) = %v, want %v", tt.content, got, tt.expected)
			}
		})
	}
}

// =====================================================
// 3. LLM 驱动摘要测试（使用真实 LLM）
// =====================================================

func TestSummarizeWithRealLLM(t *testing.T) {
	client := newRealLLMClient(t)

	t.Run("successful LLM summary with D&D content", func(t *testing.T) {
		c := llm.DefaultContextCompressor(client)
		msgs := makeDnDHistory()

		t.Logf("[LLM摘要] 开始调用 LLM 生成 D&D 结构化摘要，输入 %d 条消息...", len(msgs))
		for i, m := range msgs {
			t.Logf("[LLM摘要]   消息[%d] Role=%s, Content前50字=%.50s, ToolCalls=%d", i, m.Role, m.Content, len(m.ToolCalls))
		}
		start := time.Now()
		summary, err := c.ExportSummarizeWithLLM(context.Background(), msgs)
		elapsed := time.Since(start)
		t.Logf("[LLM摘要] 调用完成，耗时 %.1fs", elapsed.Seconds())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Logf("[LLM摘要] 摘要长度: %d 字符", len(summary))
		t.Logf("LLM Summary:\n%s", summary)
		if summary == "" {
			t.Fatal("expected non-empty summary")
		}

		summaryLower := strings.ToLower(summary)
		hasName := strings.Contains(summaryLower, "aldric") || strings.Contains(summary, "战士")
		hasEntityID := strings.Contains(summary, "01H")
		t.Logf("[LLM摘要] 验证: 包含角色名/职业=%v, 包含实体ID=%v", hasName, hasEntityID)
		if !hasName {
			t.Error("expected summary to mention the character name or class")
		}
		if !hasEntityID {
			t.Error("expected entity IDs to be preserved in summary")
		}
	})

	t.Run("summary preserves combat information", func(t *testing.T) {
		c := llm.DefaultContextCompressor(client)
		msgs := makeDnDHistory()

		t.Logf("[LLM摘要] 验证战斗信息保留，输入 %d 条消息，调用 LLM...", len(msgs))
		start := time.Now()
		summary, err := c.ExportSummarizeWithLLM(context.Background(), msgs)
		elapsed := time.Since(start)
		t.Logf("[LLM摘要] 调用完成，耗时 %.1fs, 摘要长度: %d", elapsed.Seconds(), len(summary))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		hasGoblin := strings.Contains(summary, "哥布林") || strings.Contains(strings.ToLower(summary), "goblin")
		t.Logf("[LLM摘要] 验证: 包含哥布林/goblin=%v", hasGoblin)
		t.Logf("[LLM摘要] 完整摘要:\n%s", summary)
		if !hasGoblin {
			t.Error("expected summary to mention goblin encounter")
		}
	})
}

func TestBuildSummarizationPrompt(t *testing.T) {
	c := llm.DefaultContextCompressor(nil)

	msgs := []llm.Message{
		llm.NewUserMessage("攻击哥布林"),
		llm.NewAssistantMessage("", []llm.ToolCall{
			{ID: "c1", Name: "attack", Arguments: map[string]any{"target": "goblin"}},
		}),
		{Role: llm.RoleTool, Content: `{"damage":15}`, Name: "c1"},
		llm.NewAssistantMessage("你对哥布林造成了15点伤害", nil),
		llm.NewSystemMessage("游戏状态已更新"),
	}

	prompt := c.ExportBuildSummarizationPrompt(msgs)
	t.Logf("[Prompt构建] 生成的摘要Prompt长度: %d 字符", len(prompt))
	t.Logf("[Prompt构建] Prompt内容:\n%s", prompt)

	checks := []string{"[玩家]", "[DM]", "[DM 调用工具]", "[工具结果]", "[系统]", "attack", "请压缩以下对话历史", "请按照指定格式生成结构化摘要"}
	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("expected prompt to contain %q", check)
		} else {
			t.Logf("[Prompt构建] ✓ 包含 %q", check)
		}
	}
}

func TestLLMFallbackToHeuristic(t *testing.T) {
	t.Run("nil LLMClient uses heuristic directly", func(t *testing.T) {
		t.Log("[回退] 测试 nil LLMClient 回退到启发式压缩")
		c := llm.DefaultContextCompressor(nil)
		c.RecentKeepRounds = 1
		msgs := makeHistory(3)
		result := c.CompressHistory(context.Background(), msgs)
		t.Logf("[回退] %d -> %d 条消息, 首条Role=%s", len(msgs), len(result), result[0].Role)
		t.Logf("[回退] 摘要内容前100字: %.100s", result[0].Content)
		if len(result) >= len(msgs) {
			t.Error("expected compression with nil LLMClient")
		}
		if result[0].Role != llm.RoleSystem {
			t.Errorf("expected system message, got %s", result[0].Role)
		}
		if !strings.Contains(result[0].Content, "历史上下文摘要") {
			t.Error("expected heuristic summary header")
		}
	})
}

func TestRealLLMCompressHistory(t *testing.T) {
	client := newRealLLMClient(t)

	t.Run("full compression with real LLM", func(t *testing.T) {
		c := llm.DefaultContextCompressor(client)
		c.RecentKeepRounds = 1
		msgs := makeDnDHistory()

		initTokens := c.EstimateTokens(msgs)
		t.Logf("[完整压缩] 原始消息数: %d, 估算Token: %d, RecentKeepRounds: %d", len(msgs), initTokens, c.RecentKeepRounds)
		for i, m := range msgs {
			t.Logf("[完整压缩]   原始消息[%d] Role=%s, Len=%d, ToolCalls=%d", i, m.Role, len(m.Content), len(m.ToolCalls))
		}
		t.Log("[完整压缩] 开始 CompressHistory（含 LLM 调用）...")
		start := time.Now()
		result := c.CompressHistory(context.Background(), msgs)
		elapsed := time.Since(start)
		resultTokens := c.EstimateTokens(result)
		t.Logf("[完整压缩] 完成，耗时 %.1fs", elapsed.Seconds())
		t.Logf("[完整压缩] 消息数: %d -> %d, Token: %d -> %d, 压缩率: %.1f%%",
			len(msgs), len(result), initTokens, resultTokens, float64(resultTokens)/float64(initTokens)*100)
		for i, m := range result {
			t.Logf("[完整压缩]   结果消息[%d] Role=%s, Len=%d", i, m.Role, len(m.Content))
		}

		if len(result) >= len(msgs) {
			t.Error("expected compression to reduce message count")
		}
		if result[0].Role != llm.RoleSystem {
			t.Errorf("expected system role for summary, got %s", result[0].Role)
		}
		if !strings.Contains(result[0].Content, "历史上下文摘要") {
			t.Error("expected summary header")
		}
		t.Logf("[完整压缩] Summary:\n%s", result[0].Content)
	})
}

// =====================================================
// 4. 两级压缩测试
// =====================================================

func TestPruneOldMessages(t *testing.T) {
	c := llm.DefaultContextCompressor(nil)

	t.Run("query tool results replaced with metadata", func(t *testing.T) {
		msgs := []llm.Message{
			llm.NewAssistantMessage("", []llm.ToolCall{
				{ID: "c1", Name: "get_character", Arguments: map[string]any{}},
			}),
			{Role: llm.RoleTool, Content: `{"name":"勇者","level":5,"hp":50}`, Name: "c1"},
		}
		pruned := c.ExportPruneOldMessages(msgs)
		if len(pruned) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(pruned))
		}
		toolMsg := pruned[1]
		t.Logf("[Prune] 查询工具 get_character: 原始=%d字 -> 修剪后=%d字, 内容: %s", len(msgs[1].Content), len(toolMsg.Content), toolMsg.Content)
		if !strings.Contains(toolMsg.Content, "查询结果已省略") {
			t.Errorf("expected query result to be replaced, got: %s", toolMsg.Content)
		}
	})

	t.Run("large action tool output truncated", func(t *testing.T) {
		largeContent := strings.Repeat("x", 3000)
		msgs := []llm.Message{
			llm.NewAssistantMessage("", []llm.ToolCall{
				{ID: "c1", Name: "attack", Arguments: map[string]any{}},
			}),
			{Role: llm.RoleTool, Content: largeContent, Name: "c1"},
		}
		pruned := c.ExportPruneOldMessages(msgs)
		t.Logf("[Prune] 大型动作输出 attack: 原始=%d字 -> 修剪后=%d字", len(largeContent), len(pruned[1].Content))
		if len(pruned[1].Content) >= len(largeContent) {
			t.Error("expected large content to be truncated")
		}
		if !strings.Contains(pruned[1].Content, "已截断") {
			t.Error("expected truncation marker")
		}
	})

	t.Run("small action tool output preserved", func(t *testing.T) {
		smallContent := `{"damage":15,"hit":true}`
		msgs := []llm.Message{
			llm.NewAssistantMessage("", []llm.ToolCall{
				{ID: "c1", Name: "attack", Arguments: map[string]any{}},
			}),
			{Role: llm.RoleTool, Content: smallContent, Name: "c1"},
		}
		pruned := c.ExportPruneOldMessages(msgs)
		t.Logf("[Prune] 小型动作输出 attack: 原始=%d字, 修剪后=%d字 (应保留原样)", len(smallContent), len(pruned[1].Content))
		if pruned[1].Content != smallContent {
			t.Errorf("expected small content preserved, got: %s", pruned[1].Content)
		}
	})

	t.Run("user messages preserved", func(t *testing.T) {
		msgs := []llm.Message{llm.NewUserMessage("我要攻击哥布林")}
		pruned := c.ExportPruneOldMessages(msgs)
		t.Logf("[Prune] 用户消息: 原始=%q, 修剪后=%q (应完全保留)", msgs[0].Content, pruned[0].Content)
		if pruned[0].Content != "我要攻击哥布林" {
			t.Error("expected user message to be preserved")
		}
	})
}

func TestTwoLevelCompressionWithRealLLM(t *testing.T) {
	client := newRealLLMClient(t)

	t.Run("LLM receives pruned content", func(t *testing.T) {
		t.Log("[两级压缩] 测试 Level1 Prune + Level2 LLM 摘要组合效果")
		t.Log("[两级压缩] 流程: 原始消息 -> Prune修剪 -> LLM摘要 -> 拼接最近轮次")
		c := llm.DefaultContextCompressor(client)
		c.RecentKeepRounds = 1

		msgs := []llm.Message{
			llm.NewUserMessage("查看所有角色状态"),
			llm.NewAssistantMessage("", []llm.ToolCall{
				{ID: "c1", Name: "get_character", Arguments: map[string]any{"actor_id": "01HACTOR1"}},
			}),
			{Role: llm.RoleTool, Content: strings.Repeat("detailed character data ", 50), Name: "c1"},
			llm.NewAssistantMessage("你的角色状态如上", nil),
			llm.NewUserMessage("攻击哥布林"),
			llm.NewAssistantMessage("", []llm.ToolCall{
				{ID: "c2", Name: "attack", Arguments: map[string]any{"actor_id": "01HACTOR1", "target_id": "01HGOBLIN1"}},
			}),
			{Role: llm.RoleTool, Content: `{"hit":true,"damage":15}`, Name: "c2"},
			llm.NewAssistantMessage("你攻击了哥布林", nil),
			llm.NewUserMessage("继续战斗"),
			llm.NewAssistantMessage("好的", nil),
		}

		initTokens := c.EstimateTokens(msgs)
		t.Logf("[两级压缩] 原始消息数: %d, 估算Token: %d", len(msgs), initTokens)
		for i, m := range msgs {
			t.Logf("[两级压缩]   消息[%d] Role=%s, Len=%d, ToolCalls=%d", i, m.Role, len(m.Content), len(m.ToolCalls))
		}

		// 先单独测试 Prune 效果
		pruned := c.ExportPruneOldMessages(msgs)
		prunedTokens := c.EstimateTokens(pruned)
		t.Logf("[两级压缩] Prune结果: Token %d -> %d (节省 %.1f%%)", initTokens, prunedTokens, (1-float64(prunedTokens)/float64(initTokens))*100)

		t.Log("[两级压缩] 开始 CompressHistory（Prune + LLM）...")
		start := time.Now()
		result := c.CompressHistory(context.Background(), msgs)
		elapsed := time.Since(start)
		resultTokens := c.EstimateTokens(result)
		t.Logf("[两级压缩] 完成，耗时 %.1fs", elapsed.Seconds())
		t.Logf("[两级压缩] 消息: %d -> %d, Token: %d -> %d, 总压缩率: %.1f%%",
			len(msgs), len(result), initTokens, resultTokens, float64(resultTokens)/float64(initTokens)*100)
		for i, m := range result {
			t.Logf("[两级压缩]   结果[%d] Role=%s, Len=%d", i, m.Role, len(m.Content))
		}

		if len(result) >= len(msgs) {
			t.Error("expected compression")
		}
		if result[0].Role == llm.RoleSystem {
			if strings.Contains(result[0].Content, "detailed character data") {
				t.Error("summary should not contain raw query data")
			}
			t.Logf("[两级压缩] Summary:\n%s", result[0].Content)
		}
	})
}

// =====================================================
// 5. 异步压缩测试
// =====================================================

func TestAsyncCompression(t *testing.T) {
	t.Run("start and apply compressed result", func(t *testing.T) {
		t.Log("[异步] 测试启动异步压缩并获取结果")
		c := llm.DefaultContextCompressor(nil)
		c.RecentKeepRounds = 1
		msgs := makeHistory(5)
		t.Logf("[异步] 启动异步压缩, 输入 %d 条消息", len(msgs))
		start := time.Now()
		c.StartAsyncCompress(context.Background(), msgs)

		for i := 0; i < 100; i++ {
			if !c.IsCompressing() {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Logf("[异步] 压缩完成, 耗时 %v", time.Since(start))
		result := c.ApplyCompressedIfReady()
		if result == nil {
			t.Fatal("expected compressed result")
		}
		t.Logf("[异步] 结果: %d -> %d 条消息", len(msgs), len(result))
		if len(result) >= len(msgs) {
			t.Error("compressed result should be shorter")
		}
	})

	t.Run("apply returns nil when not ready", func(t *testing.T) {
		t.Log("[异步] 测试未压缩时 Apply 返回 nil")
		c := llm.DefaultContextCompressor(nil)
		result := c.ApplyCompressedIfReady()
		t.Logf("[异步] Apply result = %v", result)
		if result != nil {
			t.Error("expected nil when no compression done")
		}
	})

	t.Run("apply clears result after retrieval", func(t *testing.T) {
		t.Log("[异步] 测试 Apply 消费后清除结果")
		c := llm.DefaultContextCompressor(nil)
		c.RecentKeepRounds = 1
		msgs := makeHistory(5)
		c.StartAsyncCompress(context.Background(), msgs)

		for i := 0; i < 100; i++ {
			if !c.IsCompressing() {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		first := c.ApplyCompressedIfReady()
		second := c.ApplyCompressedIfReady()
		t.Logf("[异步] 第一次Apply: %v (len=%d), 第二次Apply: %v", first != nil, len(first), second)
		if first == nil {
			t.Fatal("expected first apply to return result")
		}
		if second != nil {
			t.Error("expected second apply to return nil (consumed)")
		}
	})

	t.Run("no duplicate compression starts", func(t *testing.T) {
		t.Log("[异步] 测试防重复启动")
		mock := &slowMockLLMClient{delay: 200 * time.Millisecond}
		c := llm.DefaultContextCompressor(mock)
		c.RecentKeepRounds = 1
		msgs := makeHistory(5)

		t.Log("[异步] 连续调用 3 次 StartAsyncCompress")
		c.StartAsyncCompress(context.Background(), msgs)
		c.StartAsyncCompress(context.Background(), msgs)
		c.StartAsyncCompress(context.Background(), msgs)

		for i := 0; i < 50; i++ {
			if !c.IsCompressing() {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Logf("[异步] LLM 实际被调用次数: %d (期望=1)", mock.callCount.Load())
		if mock.callCount.Load() != 1 {
			t.Errorf("expected 1 LLM call, got %d", mock.callCount.Load())
		}
	})

	t.Run("IsCompressing reflects state", func(t *testing.T) {
		c := llm.DefaultContextCompressor(nil)
		t.Logf("[异步] 初始 IsCompressing=%v", c.IsCompressing())
		if c.IsCompressing() {
			t.Error("should not be compressing initially")
		}
		c.RecentKeepRounds = 1
		msgs := makeHistory(5)
		c.StartAsyncCompress(context.Background(), msgs)
		t.Logf("[异步] 启动后 IsCompressing=%v", c.IsCompressing())

		for i := 0; i < 100; i++ {
			if !c.IsCompressing() {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Logf("[异步] 完成后 IsCompressing=%v", c.IsCompressing())
		if c.IsCompressing() {
			t.Error("should not be compressing after completion")
		}
	})

	t.Run("concurrent safety", func(t *testing.T) {
		t.Log("[异步] 测试 10 个 goroutine 并发启动")
		c := llm.DefaultContextCompressor(nil)
		c.RecentKeepRounds = 1
		msgs := makeHistory(5)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.StartAsyncCompress(context.Background(), msgs)
			}()
		}
		wg.Wait()
		t.Log("[异步] 10 个 goroutine 全部完成调用")

		for i := 0; i < 100; i++ {
			if !c.IsCompressing() {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		result := c.ApplyCompressedIfReady()
		t.Logf("[异步] 并发后 Apply: result=%v", result != nil)
		if result == nil {
			t.Error("expected result after concurrent starts")
		}
	})
}

func TestAsyncCompressionWithRealLLM(t *testing.T) {
	client := newRealLLMClient(t)

	t.Run("async compress with real LLM", func(t *testing.T) {
		c := llm.DefaultContextCompressor(client)
		c.RecentKeepRounds = 1
		msgs := makeDnDHistory()

		initTokens := c.EstimateTokens(msgs)
		t.Logf("[异步LLM] 启动异步压缩，输入 %d 条消息, 估算Token: %d", len(msgs), initTokens)
		t.Logf("[异步LLM] IsCompressing=%v (启动前)", c.IsCompressing())
		start := time.Now()
		c.StartAsyncCompress(context.Background(), msgs)
		t.Logf("[异步LLM] IsCompressing=%v (启动后)", c.IsCompressing())

		// 真实 LLM 单次调用可能需要 30-50 秒，等待上限设为 120 秒
		for i := 0; i < 600; i++ {
			if !c.IsCompressing() {
				t.Logf("[异步LLM] 压缩在第 %d 次检查时完成 (%.1fs)", i+1, time.Since(start).Seconds())
				break
			}
			if (i+1)%25 == 0 {
				t.Logf("[异步LLM] 等待中... 已等待 %.0fs, IsCompressing=%v", time.Since(start).Seconds(), c.IsCompressing())
			}
			time.Sleep(200 * time.Millisecond)
		}
		t.Logf("[异步LLM] 总耗时 %.1fs", time.Since(start).Seconds())
		if c.IsCompressing() {
			t.Fatal("async compression should have completed within 120s")
		}

		result := c.ApplyCompressedIfReady()
		if result == nil {
			t.Fatal("expected compressed result from real LLM")
		}
		resultTokens := c.EstimateTokens(result)
		t.Logf("[异步LLM] 消息: %d -> %d, Token: %d -> %d, 压缩率: %.1f%%",
			len(msgs), len(result), initTokens, resultTokens, float64(resultTokens)/float64(initTokens)*100)
		for i, m := range result {
			t.Logf("[异步LLM]   结果[%d] Role=%s, Len=%d", i, m.Role, len(m.Content))
		}
		if result[0].Role != llm.RoleSystem {
			t.Errorf("expected system role, got %s", result[0].Role)
		}
		t.Logf("[异步LLM] Summary:\n%s", result[0].Content)
	})
}

// =====================================================
// 6. Token 校准测试
// =====================================================

func TestCalibrateWithActualUsage(t *testing.T) {
	t.Run("basic calibration updates ratio", func(t *testing.T) {
		c := llm.DefaultContextCompressor(nil)
		initial := c.GetCalibrationRatio()
		t.Logf("[校准] 初始ratio=%.4f, 输入: estimated=1300, actual=1000", initial)
		c.CalibrateWithActualUsage(1300, 1000)
		t.Logf("[校准] 校准后ratio=%.4f (期望变化)", c.GetCalibrationRatio())
		if c.GetCalibrationRatio() == initial {
			t.Error("expected ratio to change")
		}
	})

	t.Run("repeated calibration converges", func(t *testing.T) {
		c := llm.DefaultContextCompressor(nil)
		t.Logf("[校准] 初始ratio=%.4f, 目标收敛到~1.5", c.GetCalibrationRatio())
		for i := 0; i < 20; i++ {
			c.CalibrateWithActualUsage(int(float64(1000)*c.GetCalibrationRatio()), 1500)
			if (i+1)%5 == 0 {
				t.Logf("[校准] 第%d次: ratio=%.4f", i+1, c.GetCalibrationRatio())
			}
		}
		r := c.GetCalibrationRatio()
		t.Logf("[校准] 最终ratio=%.4f (期望1.3~1.7)", r)
		if r < 1.3 || r > 1.7 {
			t.Errorf("expected ratio near 1.5, got %f", r)
		}
	})

	t.Run("zero values ignored", func(t *testing.T) {
		c := llm.DefaultContextCompressor(nil)
		initial := c.GetCalibrationRatio()
		c.CalibrateWithActualUsage(0, 100)
		t.Logf("[校准] estimated=0: ratio %.4f -> %.4f (应不变)", initial, c.GetCalibrationRatio())
		if c.GetCalibrationRatio() != initial {
			t.Error("zero estimated should not change ratio")
		}
		c.CalibrateWithActualUsage(100, 0)
		t.Logf("[校准] actual=0: ratio %.4f -> %.4f (应不变)", initial, c.GetCalibrationRatio())
		if c.GetCalibrationRatio() != initial {
			t.Error("zero actual should not change ratio")
		}
	})

	t.Run("calibration improves estimation", func(t *testing.T) {
		c := llm.DefaultContextCompressor(nil)
		msgs := makeHistory(10)
		initialEst := c.EstimateTokens(msgs)
		actual := int(float64(initialEst) * 0.8)
		t.Logf("[校准] 初始估算=%d, 实际=%d (偏高20%%)", initialEst, actual)
		c.CalibrateWithActualUsage(initialEst, actual)
		calibrated := c.EstimateTokens(msgs)
		t.Logf("[校准] 校准后估算=%d (期望 < %d)", calibrated, initialEst)
		if calibrated >= initialEst {
			t.Error("calibrated estimate should be lower")
		}
	})
}

// =====================================================
// 7. 边界条件测试
// =====================================================

func TestEdgeCases(t *testing.T) {
	t.Run("compress nil messages", func(t *testing.T) {
		t.Log("[边界] 测试压缩 nil 消息")
		c := llm.DefaultContextCompressor(nil)
		result := c.CompressHistory(context.Background(), nil)
		t.Logf("[边界] CompressHistory(nil) = %v", result)
		if result != nil {
			t.Error("expected nil for nil messages")
		}
	})

	t.Run("compress empty slice", func(t *testing.T) {
		t.Log("[边界] 测试压缩空切片")
		c := llm.DefaultContextCompressor(nil)
		result := c.CompressHistory(context.Background(), []llm.Message{})
		t.Logf("[边界] CompressHistory([]) = %v (len=%d)", result, len(result))
		if result != nil && len(result) != 0 {
			t.Error("expected empty result")
		}
	})

	t.Run("single message not compressed", func(t *testing.T) {
		t.Log("[边界] 测试单条消息不压缩")
		c := llm.DefaultContextCompressor(nil)
		c.RecentKeepRounds = 1
		msgs := []llm.Message{llm.NewUserMessage("hello")}
		result := c.CompressHistory(context.Background(), msgs)
		t.Logf("[边界] 单条消息: 输入=%d, 输出=%d", len(msgs), len(result))
		if len(result) != 1 {
			t.Errorf("expected 1 message, got %d", len(result))
		}
	})

	t.Run("nil LLMClient fallback", func(t *testing.T) {
		t.Log("[边界] 测试 nil LLMClient 回退")
		c := llm.DefaultContextCompressor(nil)
		c.RecentKeepRounds = 1
		msgs := makeHistory(5)
		result := c.CompressHistory(context.Background(), msgs)
		t.Logf("[边界] nil LLMClient: %d -> %d 条消息", len(msgs), len(result))
		if len(result) >= len(msgs) {
			t.Error("expected compression")
		}
	})

	t.Run("prune empty messages", func(t *testing.T) {
		t.Log("[边界] 测试 Prune nil 消息")
		c := llm.DefaultContextCompressor(nil)
		result := c.ExportPruneOldMessages(nil)
		t.Logf("[边界] PruneOldMessages(nil) len=%d", len(result))
		if len(result) != 0 {
			t.Error("expected empty result")
		}
	})

	t.Run("heuristic fallback for empty messages", func(t *testing.T) {
		t.Log("[边界] 测试启发式压缩 nil 消息")
		c := llm.DefaultContextCompressor(nil)
		result := c.ExportCompressOldMessagesHeuristic(nil)
		t.Logf("[边界] CompressOldMessagesHeuristic(nil) = %v", result)
		if result != nil {
			t.Error("expected nil")
		}
	})

	t.Run("large message count performance", func(t *testing.T) {
		c := llm.DefaultContextCompressor(nil)
		c.RecentKeepRounds = 2
		c.SetCalibrationRatio(1.0)
		msgs := makeHistory(100)

		initTokens := c.EstimateTokens(msgs)
		t.Logf("[性能] 压缩 %d 条消息（100轮），估算Token: %d, RecentKeepRounds=%d", len(msgs), initTokens, c.RecentKeepRounds)
		start := time.Now()
		result := c.CompressHistory(context.Background(), msgs)
		elapsed := time.Since(start)
		resultTokens := c.EstimateTokens(result)
		t.Logf("[性能] 完成，耗时 %v", elapsed)
		t.Logf("[性能] 消息: %d -> %d, Token: %d -> %d, 压缩率: %.1f%%",
			len(msgs), len(result), initTokens, resultTokens, float64(resultTokens)/float64(initTokens)*100)
		t.Logf("[性能] 结果首条 Role=%s, 末条 Role=%s", result[0].Role, result[len(result)-1].Role)

		if elapsed > 2*time.Second {
			t.Errorf("took too long: %v", elapsed)
		}
		if len(result) >= len(msgs) {
			t.Error("expected compression for 100 rounds")
		}
	})

	t.Run("estimateStringTokens edge cases", func(t *testing.T) {
		emptyTokens := llm.ExportEstimateStringTokens("")
		singleTokens := llm.ExportEstimateStringTokens("a")
		longTokens := llm.ExportEstimateStringTokens(strings.Repeat("hello world ", 100))
		t.Logf("[Token边界] empty=%d, single='a'=%d, long(1200字)=%d", emptyTokens, singleTokens, longTokens)
		if emptyTokens != 0 {
			t.Error("empty string should be 0 tokens")
		}
		if singleTokens <= 0 {
			t.Error("single char should be > 0 tokens")
		}
	})
}
