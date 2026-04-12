package agent

import (
	"context"

	"github.com/zwh8800/dnd-core/pkg/engine"

	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/state"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// Agent 基础接口
type Agent interface {
	// Name 返回Agent名称
	Name() string

	// Description 返回Agent描述
	Description() string

	// SystemPrompt 返回系统提示词
	SystemPrompt(ctx *AgentContext) string

	// Tools 返回Agent可用的Tools
	Tools() []tool.Tool

	// Execute 执行Agent逻辑
	Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
}

// SubAgent 子Agent接口
type SubAgent interface {
	Agent

	// CanHandle 判断是否能处理该意图
	CanHandle(intent string) bool

	// Priority 返回处理优先级
	Priority() int

	// Dependencies 返回依赖的其他Agent
	Dependencies() []string
}

// AgentContext Agent执行上下文
type AgentContext struct {
	GameID       string             // 游戏会话ID
	PlayerID     string             // 玩家角色ID
	Engine       *engine.Engine     // D&D引擎实例
	History      []llm.Message      // 对话历史
	CurrentState *state.GameSummary // 当前状态摘要
	Metadata     map[string]any     // 扩展元数据
}

// AgentRequest Agent请求
type AgentRequest struct {
	UserInput       string                    // 用户输入
	Intent          string                    // 解析后的意图
	Context         *AgentContext             // 执行上下文
	SubAgentResults map[string]*AgentResponse // 子Agent返回结果
}

// SubAgentCall 子Agent调用请求
type SubAgentCall struct {
	AgentName string `json:"agent_name"` // 子Agent名称，如 "character_agent"
	Intent    string `json:"intent"`     // 传入子Agent的意图描述
}

// AgentResponse Agent响应
type AgentResponse struct {
	Content       string         `json:"content"`         // 生成的文本内容
	ToolCalls     []llm.ToolCall `json:"tool_calls"`      // 需要执行的Tool调用
	SubAgentCalls []SubAgentCall `json:"sub_agent_calls"` // 需要调用的子Agent
	NextAction    NextAction     `json:"next_action"`     // 下一步动作
	StateChange   *StateChange   `json:"state_change"`    // 状态变更
	Errors        []AgentError   `json:"errors"`          // 错误信息
}

// NextAction 下一步动作类型
type NextAction int

const (
	ActionContinue        NextAction = iota // 继续思考
	ActionCallSubAgent                      // 调用子Agent
	ActionRespondToPlayer                   // 响应玩家
	ActionWaitForInput                      // 等待玩家输入
	ActionEndGame                           // 结束游戏
)

// StateChange 状态变更
type StateChange struct {
	Type    string `json:"type"`
	Details any    `json:"details"`
}

// AgentError Agent错误
type AgentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewAgentContext 创建新的Agent上下文
func NewAgentContext(gameID, playerID string, engine *engine.Engine) *AgentContext {
	return &AgentContext{
		GameID:   gameID,
		PlayerID: playerID,
		Engine:   engine,
		History:  make([]llm.Message, 0),
		Metadata: make(map[string]any),
	}
}

// AddHistory 添加消息到历史
func (c *AgentContext) AddHistory(msg llm.Message) {
	c.History = append(c.History, msg)
}

// GetMetadata 获取元数据
func (c *AgentContext) GetMetadata(key string) (any, bool) {
	val, ok := c.Metadata[key]
	return val, ok
}

// SetMetadata 设置元数据
func (c *AgentContext) SetMetadata(key string, value any) {
	c.Metadata[key] = value
}
