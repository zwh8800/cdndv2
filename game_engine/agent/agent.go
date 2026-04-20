package agent

import (
	"context"

	"github.com/zwh8800/dnd-core/pkg/engine"

	"github.com/zwh8800/cdndv2/game_engine/game_summary"
	"github.com/zwh8800/cdndv2/game_engine/llm"
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
	GameID       string                    // 游戏会话ID
	PlayerID     string                    // 玩家角色ID
	Engine       *engine.Engine            // D&D引擎实例
	History      []llm.Message             // 对话历史
	CurrentState *game_summary.GameSummary // 当前状态摘要
	Metadata     map[string]any            // 扩展元数据

	// SubAgent 执行相关
	AgentResults   map[string]*AgentCallResult // SubAgent 执行结果
	Parent         *AgentContext               // 父会话引用（SubAgent隔离用）
	IsSubSession   bool                        // 是否为子会话
	KnownEntityIDs map[string]string            // 已知实体ID映射（如 actor_id, scene_id 等），用于 SubAgent 间共享
}

// AgentRequest Agent请求
type AgentRequest struct {
	UserInput       string                    // 用户输入
	Intent          string                    // 解析后的意图
	Context         *AgentContext             // 执行上下文
	SubAgentResults map[string]*AgentResponse // 子Agent返回结果
}

// AgentCallResult Agent 执行结果
type AgentCallResult struct {
	AgentName string         `json:"agent_name"`
	Success   bool           `json:"success"`
	Content   string         `json:"content"`
	ToolCalls []llm.ToolCall `json:"tool_calls,omitempty"`
	State     map[string]any `json:"state,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// ExecutionMode 执行模式
type ExecutionMode string

const (
	ExecutionSequential ExecutionMode = "sequential"
	ExecutionParallel   ExecutionMode = "parallel"
)

// AgentDelegation 单个Agent委托请求
type AgentDelegation struct {
	AgentName string `json:"agent_name"`
	Intent    string `json:"intent"`
	Input     string `json:"input,omitempty"`
}

// RouterDecision 路由决策
type RouterDecision struct {
	TargetAgents   []AgentDelegation `json:"target_agents"`
	ExecutionMode  ExecutionMode     `json:"execution_mode"`
	Reasoning      string            `json:"reasoning"`
	DirectResponse string            `json:"direct_response,omitempty"` // 无需Agent时的直接回复
}

// SubAgentCall 子Agent调用请求（已废弃，保留兼容，使用 AgentDelegation 替代）
type SubAgentCall struct {
	AgentName string `json:"agent_name"` // 子Agent名称，如 "character_agent"
	Intent    string `json:"intent"`     // 传入子Agent的意图描述
}

// AgentResponse Agent响应
type AgentResponse struct {
	Content           string         `json:"content"`             // 生成的文本内容
	ToolCalls         []llm.ToolCall `json:"tool_calls"`          // 需要执行的Tool调用
	DelegateToolCalls []llm.ToolCall `json:"delegate_tool_calls"` // 原始 delegate_task 工具调用（用于维护 OpenAI 对话格式）
	SubAgentCalls     []SubAgentCall `json:"sub_agent_calls"`     // 需要调用的子Agent
	NextAction        NextAction     `json:"next_action"`         // 下一步动作
	StateChange       *StateChange   `json:"state_change"`        // 状态变更
	Errors            []AgentError   `json:"errors"`              // 错误信息
}

// NextAction 下一步动作类型
type NextAction int

const (
	ActionContinue        NextAction = iota // 继续思考
	ActionDelegate                          // 委托给SubAgent
	ActionSynthesize                        // 合成结果
	ActionRespondToPlayer                   // 响应玩家
	ActionWaitForInput                      // 等待玩家输入
	ActionEndGame                           // 结束游戏
)

// String 返回 NextAction 的字符串表示
func (a NextAction) String() string {
	switch a {
	case ActionContinue:
		return "Continue"
	case ActionDelegate:
		return "Delegate"
	case ActionSynthesize:
		return "Synthesize"
	case ActionRespondToPlayer:
		return "RespondToPlayer"
	case ActionWaitForInput:
		return "WaitForInput"
	case ActionEndGame:
		return "EndGame"
	default:
		return "Unknown"
	}
}

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

// AddAgentResult 添加 SubAgent 执行结果
func (c *AgentContext) AddAgentResult(result *AgentCallResult) {
	if c.AgentResults == nil {
		c.AgentResults = make(map[string]*AgentCallResult)
	}
	c.AgentResults[result.AgentName] = result
}

// GetAgentResult 获取特定 SubAgent 的结果
func (c *AgentContext) GetAgentResult(agentName string) *AgentCallResult {
	if c.AgentResults == nil {
		return nil
	}
	return c.AgentResults[agentName]
}

// GetAllAgentResults 获取所有 SubAgent 结果
func (c *AgentContext) GetAllAgentResults() map[string]*AgentCallResult {
	return c.AgentResults
}

// HasAgentResults 检查是否有 SubAgent 结果
func (c *AgentContext) HasAgentResults() bool {
	return c.AgentResults != nil && len(c.AgentResults) > 0
}

// ClearAgentResults 清除 SubAgent 结果
func (c *AgentContext) ClearAgentResults() {
	c.AgentResults = nil
}

// SetKnownEntityID 设置已知实体ID
func (c *AgentContext) SetKnownEntityID(entityType, entityID string) {
	if c.KnownEntityIDs == nil {
		c.KnownEntityIDs = make(map[string]string)
	}
	c.KnownEntityIDs[entityType] = entityID
}

// GetKnownEntityID 获取已知实体ID
func (c *AgentContext) GetKnownEntityID(entityType string) (string, bool) {
	if c.KnownEntityIDs == nil {
		return "", false
	}
	id, ok := c.KnownEntityIDs[entityType]
	return id, ok
}

// MergeKnownEntityIDs 合并已知实体ID（从其他上下文或结果中）
func (c *AgentContext) MergeKnownEntityIDs(other map[string]string) {
	if c.KnownEntityIDs == nil {
		c.KnownEntityIDs = make(map[string]string)
	}
	for k, v := range other {
		c.KnownEntityIDs[k] = v
	}
}
