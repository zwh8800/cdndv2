package gameengine

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"

	"github.com/zwh8800/cdndv2/game_engine/agent"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/state"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// Phase ReAct循环阶段
type Phase int

const (
	PhaseObserve Phase = iota
	PhaseThink
	PhaseAct
	PhaseWait
	PhaseEnd
)

// LoopState 循环状态
type LoopState struct {
	GameID       model.ID
	PlayerID     model.ID
	History      []llm.Message
	CurrentPhase Phase
	Iteration    int
	LastResult   *agent.AgentResponse
	agentContext *agent.AgentContext
}

// ReActLoop ReAct循环控制器
type ReActLoop struct {
	engine    *engine.Engine
	mainAgent agent.Agent
	agents    map[string]agent.SubAgent
	tools     *tool.ToolRegistry
	llm       llm.LLMClient
	state     *LoopState
	maxIter   int
}

// NewReActLoop 创建ReAct循环控制器
func NewReActLoop(
	e *engine.Engine,
	mainAgent agent.Agent,
	agents map[string]agent.SubAgent,
	tools *tool.ToolRegistry,
	llmClient llm.LLMClient,
	maxIter int,
) *ReActLoop {
	return &ReActLoop{
		engine:    e,
		mainAgent: mainAgent,
		agents:    agents,
		tools:     tools,
		llm:       llmClient,
		maxIter:   maxIter,
		state: &LoopState{
			CurrentPhase: PhaseObserve,
			History:      make([]llm.Message, 0),
		},
	}
}

// Run 执行ReAct循环
func (l *ReActLoop) Run(ctx context.Context, initialInput string, gameID, playerID model.ID) error {
	l.state.GameID = gameID
	l.state.PlayerID = playerID
	l.state.Iteration = 0
	l.state.CurrentPhase = PhaseObserve

	for l.state.CurrentPhase != PhaseEnd {
		if l.state.Iteration >= l.maxIter {
			return fmt.Errorf("max iterations reached (%d)", l.maxIter)
		}

		switch l.state.CurrentPhase {
		case PhaseObserve:
			l.observe(ctx)

		case PhaseThink:
			response, err := l.think(ctx)
			if err != nil {
				return fmt.Errorf("think phase failed: %w", err)
			}
			l.state.LastResult = response
			l.state.CurrentPhase = PhaseAct

		case PhaseAct:
			nextPhase := l.act(ctx)
			l.state.CurrentPhase = nextPhase

		case PhaseWait:
			// 等待玩家输入（由外部处理）
			return nil
		}

		l.state.Iteration++
	}

	return nil
}

// observe 观察阶段
func (l *ReActLoop) observe(ctx context.Context) {
	// 收集游戏状态
	summary, err := state.CollectSummary(ctx, l.engine, l.state.GameID, l.state.PlayerID)
	if err != nil {
		// 如果收集失败，创建空摘要
		summary = state.NewGameSummary(l.state.GameID)
	}

	// 构建上下文
	l.state.agentContext = agent.NewAgentContext(
		string(l.state.GameID),
		string(l.state.PlayerID),
		l.engine,
	)
	l.state.agentContext.History = l.state.History
	l.state.agentContext.CurrentState = summary

	// 转入思考阶段
	l.state.CurrentPhase = PhaseThink
}

// think 思考阶段
func (l *ReActLoop) think(ctx context.Context) (*agent.AgentResponse, error) {
	req := &agent.AgentRequest{
		Context: l.state.agentContext,
	}

	// 获取最近的玩家输入
	if len(l.state.History) > 0 {
		lastMsg := l.state.History[len(l.state.History)-1]
		if lastMsg.Role == llm.RoleUser {
			req.UserInput = lastMsg.Content
		}
	}

	return l.mainAgent.Execute(ctx, req)
}

// act 行动阶段
func (l *ReActLoop) act(ctx context.Context) Phase {
	result := l.state.LastResult
	if result == nil {
		return PhaseWait
	}

	// 处理Tool调用
	if len(result.ToolCalls) > 0 {
		toolResults := l.executeTools(ctx, result.ToolCalls)

		// 将结果添加到历史
		for _, tr := range toolResults {
			l.state.History = append(l.state.History, llm.NewToolMessage(tr.Content, tr.ToolCallID))
		}

		return PhaseThink // 继续思考
	}

	// 处理内容输出
	if result.Content != "" {
		l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, nil))
	}

	// 根据下一步动作决定
	switch result.NextAction {
	case agent.ActionCallSubAgent:
		subAgentResults := l.executeSubAgents(ctx, result.SubAgentCalls)
		if len(subAgentResults) > 0 {
			// 将子Agent结果传递给主Agent，让主Agent继续处理
			l.state.History = append(l.state.History, llm.NewAssistantMessage(
				fmt.Sprintf("子Agent执行完成，共%d个结果", len(subAgentResults)),
				nil,
			))
			// 将子Agent结果存入上下文，供主Agent下一轮使用
			if l.state.agentContext != nil {
				l.state.agentContext.Metadata["sub_agent_results"] = subAgentResults
			}
		}
		return PhaseThink // 主Agent继续思考

	case agent.ActionWaitForInput:
		return PhaseWait
	case agent.ActionEndGame:
		return PhaseEnd
	case agent.ActionRespondToPlayer:
		return PhaseWait
	default:
		return PhaseWait
	}
}

// executeSubAgents 执行子Agent调用
func (l *ReActLoop) executeSubAgents(ctx context.Context, calls []agent.SubAgentCall) map[string]*agent.AgentResponse {
	results := make(map[string]*agent.AgentResponse)

	for _, call := range calls {
		subAgent, ok := l.agents[call.AgentName]
		if !ok {
			results[call.AgentName] = &agent.AgentResponse{
				Content: "错误: 未找到子Agent " + call.AgentName,
				Errors:  []agent.AgentError{{Code: "AGENT_NOT_FOUND", Message: "agent not found"}},
			}
			continue
		}

		// 构建子Agent请求
		req := &agent.AgentRequest{
			UserInput: call.Intent,
			Context:   l.state.agentContext,
		}

		// 执行子Agent
		resp, err := subAgent.Execute(ctx, req)
		if err != nil {
			results[call.AgentName] = &agent.AgentResponse{
				Content: "错误: 子Agent执行失败: " + err.Error(),
				Errors:  []agent.AgentError{{Code: "EXECUTION_ERROR", Message: err.Error()}},
			}
			continue
		}

		results[call.AgentName] = resp

		// 将子Agent的输出添加到对话历史
		if resp.Content != "" {
			l.state.History = append(l.state.History, llm.NewToolMessage(
				fmt.Sprintf("[%s] %s", subAgent.Name(), resp.Content),
				call.AgentName,
			))
		}

		// 如果子Agent有Tool调用，也执行它们
		if len(resp.ToolCalls) > 0 {
			toolResults := l.executeTools(ctx, resp.ToolCalls)
			for _, tr := range toolResults {
				l.state.History = append(l.state.History, llm.NewToolMessage(tr.Content, tr.ToolCallID))
			}
		}
	}

	return results
}

// executeTools 执行Tools
func (l *ReActLoop) executeTools(ctx context.Context, calls []llm.ToolCall) []llm.ToolResult {
	return l.tools.ExecuteTools(ctx, calls)
}

// GetHistory 获取对话历史
func (l *ReActLoop) GetHistory() []llm.Message {
	return l.state.History
}

// GetState 获取循环状态
func (l *ReActLoop) GetState() *LoopState {
	return l.state
}
