package gameengine

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
	"go.uber.org/zap"

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
	logger    *zap.Logger
}

// getLogger 获取日志器
func (l *ReActLoop) getLogger() *zap.Logger {
	if l.logger == nil {
		l.logger = zap.NewNop()
	}
	return l.logger
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
		logger: zap.NewNop(),
	}
}

// SetLogger 设置日志器
func (l *ReActLoop) SetLogger(log *zap.Logger) {
	if log != nil {
		l.logger = log
	}
}

// Run 执行ReAct循环
func (l *ReActLoop) Run(ctx context.Context, initialInput string, gameID, playerID model.ID) error {
	log := l.getLogger()

	l.state.GameID = gameID
	l.state.PlayerID = playerID
	l.state.Iteration = 0
	l.state.CurrentPhase = PhaseObserve

	log.Info("ReAct Loop started",
		zap.String("gameID", string(gameID)),
		zap.String("playerID", string(playerID)),
		zap.String("initialInput", initialInput),
		zap.Int("maxIterations", l.maxIter),
	)

	for l.state.CurrentPhase != PhaseEnd {
		if l.state.Iteration >= l.maxIter {
			log.Warn("Max iterations reached",
				zap.Int("iteration", l.state.Iteration),
				zap.Int("maxIter", l.maxIter),
			)
			return fmt.Errorf("max iterations reached (%d)", l.maxIter)
		}

		log.Debug("Loop iteration",
			zap.Int("iteration", l.state.Iteration),
			zap.String("phase", l.phaseName(l.state.CurrentPhase)),
		)

		switch l.state.CurrentPhase {
		case PhaseObserve:
			l.observe(ctx)

		case PhaseThink:
			response, err := l.think(ctx)
			if err != nil {
				log.Error("Think phase failed",
					zap.Int("iteration", l.state.Iteration),
					zap.Error(err),
				)
				return fmt.Errorf("think phase failed: %w", err)
			}
			l.state.LastResult = response
			l.state.CurrentPhase = PhaseAct

		case PhaseAct:
			nextPhase := l.act(ctx)
			l.state.CurrentPhase = nextPhase

		case PhaseWait:
			// 等待玩家输入（由外部处理）
			log.Debug("Waiting for player input",
				zap.Int("iteration", l.state.Iteration),
			)
			return nil
		}

		l.state.Iteration++
	}

	log.Info("ReAct Loop completed",
		zap.Int("totalIterations", l.state.Iteration),
		zap.Int("historyLength", len(l.state.History)),
	)

	return nil
}

// phaseName 获取阶段名称
func (l *ReActLoop) phaseName(phase Phase) string {
	switch phase {
	case PhaseObserve:
		return "Observe"
	case PhaseThink:
		return "Think"
	case PhaseAct:
		return "Act"
	case PhaseWait:
		return "Wait"
	case PhaseEnd:
		return "End"
	default:
		return "Unknown"
	}
}

// observe 观察阶段
func (l *ReActLoop) observe(ctx context.Context) {
	log := l.getLogger()
	log.Debug("Observe phase started")

	// 收集游戏状态
	summary, err := state.CollectSummary(ctx, l.engine, l.state.GameID, l.state.PlayerID)
	if err != nil {
		log.Warn("Failed to collect summary, using empty summary",
			zap.Error(err),
			zap.String("gameID", string(l.state.GameID)),
		)
		summary = state.NewGameSummary(l.state.GameID)
	}

	log.Debug("Game summary collected",
		zap.Int("historyCount", len(l.state.History)),
		zap.Any("gameState", summary),
	)

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
	log.Debug("Transitioning to Think phase")
}

// think 思考阶段
func (l *ReActLoop) think(ctx context.Context) (*agent.AgentResponse, error) {
	log := l.getLogger()
	log.Debug("Think phase started")

	req := &agent.AgentRequest{
		Context: l.state.agentContext,
	}

	// 获取最近的玩家输入
	if len(l.state.History) > 0 {
		lastMsg := l.state.History[len(l.state.History)-1]
		if lastMsg.Role == llm.RoleUser {
			req.UserInput = lastMsg.Content
			log.Debug("User input extracted from history",
				zap.String("userInput", lastMsg.Content),
			)
		}
	}

	log.Debug("Calling MainAgent.Execute",
		zap.String("userInput", req.UserInput),
		zap.Int("historyLength", len(req.Context.History)),
	)

	resp, err := l.mainAgent.Execute(ctx, req)
	if err != nil {
		log.Error("MainAgent.Execute failed",
			zap.Error(err),
		)
		return nil, err
	}

	log.Debug("MainAgent.Execute completed",
		zap.String("content", truncateString(resp.Content, 200)),
		zap.Int("toolCalls", len(resp.ToolCalls)),
		zap.Int("subAgentCalls", len(resp.SubAgentCalls)),
		zap.String("nextAction", resp.NextAction.String()),
	)

	return resp, nil
}

// act 行动阶段
func (l *ReActLoop) act(ctx context.Context) Phase {
	log := l.getLogger()
	log.Debug("Act phase started")

	result := l.state.LastResult
	if result == nil {
		log.Debug("No result from think phase, returning PhaseWait")
		return PhaseWait
	}

	// 处理Tool调用
	if len(result.ToolCalls) > 0 {
		log.Info("Executing tool calls",
			zap.Int("count", len(result.ToolCalls)),
		)

		for i, tc := range result.ToolCalls {
			argsJSON := formatArgs(tc.Arguments)
			log.Debug("Tool call",
				zap.Int("index", i),
				zap.String("toolName", tc.Name),
				zap.String("toolCallID", tc.ID),
				zap.Any("arguments", argsJSON),
			)
		}

		toolResults := l.executeTools(ctx, result.ToolCalls)

		// 将结果添加到历史
		for _, tr := range toolResults {
			l.state.History = append(l.state.History, llm.NewToolMessage(tr.Content, tr.ToolCallID))
			log.Debug("Tool result added to history",
				zap.String("toolCallID", tr.ToolCallID),
				zap.Bool("isError", tr.IsError),
				zap.String("content", truncateString(tr.Content, 100)),
			)
		}

		return PhaseThink // 继续思考
	}

	// 处理内容输出
	if result.Content != "" {
		l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, nil))
		log.Debug("Assistant message added to history",
			zap.String("content", truncateString(result.Content, 200)),
		)
	}

	// 根据下一步动作决定
	switch result.NextAction {
	case agent.ActionCallSubAgent:
		log.Info("Action: CallSubAgent",
			zap.Int("subAgentCount", len(result.SubAgentCalls)),
		)
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
		log.Info("Action: WaitForInput")
		return PhaseWait
	case agent.ActionEndGame:
		log.Info("Action: EndGame")
		return PhaseEnd
	case agent.ActionRespondToPlayer:
		log.Info("Action: RespondToPlayer",
			zap.String("content", truncateString(result.Content, 200)),
		)
		return PhaseWait
	default:
		log.Info("Action: default (WaitForInput)")
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

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatArgs 格式化参数为可读字符串
func formatArgs(args map[string]any) string {
	if args == nil {
		return "{}"
	}
	s := "{"
	first := true
	for k, v := range args {
		if !first {
			s += ", "
		}
		first = false
		s += fmt.Sprintf("%s=%v", k, v)
	}
	s += "}"
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
