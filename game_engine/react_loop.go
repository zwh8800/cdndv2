package gameengine

import (
	"context"
	"fmt"
	"sync"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/zwh8800/cdndv2/game_engine/agent"
	"github.com/zwh8800/cdndv2/game_engine/game_summary"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// Phase ReAct循环阶段
type Phase int

const (
	PhaseObserve Phase = iota
	PhaseRoute
	PhaseThink
	PhaseAct
	PhaseSynthesize
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
	router    *agent.RouterAgent
	agents    map[string]agent.SubAgent
	tools     *tool.ToolRegistry
	llm       llm.LLMClient
	state     *LoopState
	maxIter   int
	useRouter bool
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
	router *agent.RouterAgent,
	agents map[string]agent.SubAgent,
	tools *tool.ToolRegistry,
	llmClient llm.LLMClient,
	maxIter int,
) *ReActLoop {
	return &ReActLoop{
		engine:    e,
		mainAgent: mainAgent,
		router:    router,
		agents:    agents,
		tools:     tools,
		llm:       llmClient,
		maxIter:   maxIter,
		useRouter: router != nil,
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

	log.Debug("ReAct Loop started",
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

		case PhaseRoute:
			l.route(ctx)

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

		case PhaseSynthesize:
			l.synthesize(ctx)

		case PhaseWait:
			// 等待玩家输入（由外部处理）
			log.Debug("Waiting for player input",
				zap.Int("iteration", l.state.Iteration),
			)
			return nil
		}

		l.state.Iteration++
	}

	log.Debug("ReAct Loop completed",
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
	case PhaseRoute:
		return "Route"
	case PhaseThink:
		return "Think"
	case PhaseAct:
		return "Act"
	case PhaseSynthesize:
		return "Synthesize"
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
	summary, err := game_summary.CollectSummary(ctx, l.engine, l.state.GameID, l.state.PlayerID)
	if err != nil {
		log.Warn("Failed to collect summary, using empty summary",
			zap.Error(err),
			zap.String("gameID", string(l.state.GameID)),
		)
		summary = game_summary.NewGameSummary(l.state.GameID)
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

	// 如果启用了Router，转入路由阶段；否则直接转入思考阶段
	if l.useRouter {
		l.state.CurrentPhase = PhaseRoute
		log.Debug("Transitioning to Route phase")
	} else {
		l.state.CurrentPhase = PhaseThink
		log.Debug("Transitioning to Think phase")
	}
}

// route 路由阶段
func (l *ReActLoop) route(ctx context.Context) {
	log := l.getLogger()
	log.Debug("Route phase started")

	// 获取用户输入
	var userInput string
	if len(l.state.History) > 0 {
		lastMsg := l.state.History[len(l.state.History)-1]
		if lastMsg.Role == llm.RoleUser {
			userInput = lastMsg.Content
		}
	}

	// 调用RouterAgent进行路由决策
	decision, err := l.router.Route(ctx, userInput, l.state.History, l.state.agentContext.CurrentState)
	if err != nil {
		log.Error("Router failed, falling back to Think phase",
			zap.Error(err),
		)
		l.state.CurrentPhase = PhaseThink
		return
	}

	log.Debug("Router decision",
		zap.Int("targetAgentCount", len(decision.TargetAgents)),
		zap.String("executionMode", string(decision.ExecutionMode)),
		zap.String("reasoning", decision.Reasoning),
	)

	// 如果有直接响应，跳转到Think阶段让MainAgent处理
	if decision.DirectResponse != "" && len(decision.TargetAgents) == 0 {
		log.Debug("Router decided direct response, going to Think phase")
		l.state.CurrentPhase = PhaseThink
		return
	}

	// 如果有目标Agent，执行委托
	if len(decision.TargetAgents) > 0 {
		// 转换为SubAgentCall格式
		calls := make([]agent.SubAgentCall, 0, len(decision.TargetAgents))
		for _, t := range decision.TargetAgents {
			calls = append(calls, agent.SubAgentCall{
				AgentName: t.AgentName,
				Intent:    t.Intent,
			})
		}

		// 执行委托（根据决策中的执行模式）
		results := l.executeDelegations(ctx, calls)

		// 将结果存入上下文
		if l.state.agentContext != nil {
			for _, res := range results {
				l.state.agentContext.AddAgentResult(res)
			}
		}

		// 生成结果摘要添加到历史
		var resultSummaries []string
		for agentName, res := range results {
			if res.Success {
				resultSummaries = append(resultSummaries, fmt.Sprintf("[%s] %s", agentName, res.Content))
			} else {
				resultSummaries = append(resultSummaries, fmt.Sprintf("[%s] 错误: %s", agentName, res.Error))
			}
		}
		if len(resultSummaries) > 0 {
			l.state.History = append(l.state.History, llm.NewAssistantMessage(
				fmt.Sprintf("委托任务执行完成:\n%s", joinStrings(resultSummaries, "\n")),
				nil,
			))
		}

		// 同步 agentContext.History，确保 Think 阶段能看到完整历史
		if l.state.agentContext != nil {
			l.state.agentContext.History = l.state.History
		}
	}

	// 转入Think阶段，让MainAgent处理结果或生成叙事
	l.state.CurrentPhase = PhaseThink
	log.Debug("Transitioning to Think phase after routing")
}

// think 思考阶段
func (l *ReActLoop) think(ctx context.Context) (*agent.AgentResponse, error) {
	log := l.getLogger()
	log.Debug("Think phase started")

	req := &agent.AgentRequest{
		Context: l.state.agentContext,
	}

	// 获取最近的玩家输入（从 history 最后一条提取）
	// 注意：这条消息已经存在于 history 中，不需要再通过 UserInput 添加
	if len(l.state.History) > 0 {
		lastMsg := l.state.History[len(l.state.History)-1]
		if lastMsg.Role == llm.RoleUser {
			req.UserInput = lastMsg.Content
			log.Debug("User input extracted from history",
				zap.String("userInput", lastMsg.Content),
			)
		}
	}

	// UserInput 已从 history 提取，不能再作为新消息添加，否则会重复
	// 方案：在 agentContext 中标记，避免重复添加
	if l.state.agentContext != nil {
		l.state.agentContext.Metadata["pending_user_input"] = req.UserInput
		req.UserInput = "" // 清空，让 buildMessages 不要重复添加
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
		log.Debug("Executing tool calls",
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

		// 先将 assistant 消息（包含 tool_calls）加入历史
		// OpenAI function calling 格式要求：assistant 消息在前，tool result 消息在后
		l.state.History = append(l.state.History, llm.NewAssistantMessage("", result.ToolCalls))
		log.Debug("Assistant tool call message added to history",
			zap.Int("toolCallCount", len(result.ToolCalls)),
		)

		toolResults := l.executeTools(ctx, result.ToolCalls)

		// 将 tool 结果添加到历史
		for _, tr := range toolResults {
			l.state.History = append(l.state.History, llm.NewToolMessage(tr.Content, tr.ToolCallID))
			log.Debug("Tool result added to history",
				zap.String("toolCallID", tr.ToolCallID),
				zap.Bool("isError", tr.IsError),
				zap.String("content", truncateString(tr.Content, 100)),
			)
		}

		// 同步 agentContext.History，确保下一轮 LLM 能看到 tool 结果
		if l.state.agentContext != nil {
			l.state.agentContext.History = l.state.History
		}

		return PhaseThink // 继续思考
	}

	// 处理内容输出（ActionDelegate 由其分支内以 tool_call 格式添加，避免重复）
	if result.Content != "" && result.NextAction != agent.ActionDelegate {
		l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, nil))
		log.Debug("Assistant message added to history",
			zap.String("content", truncateString(result.Content, 200)),
		)
	}

	// 根据下一步动作决定
	switch result.NextAction {
	case agent.ActionDelegate:
		log.Debug("Action: Delegate",
			zap.Int("delegationCount", len(result.SubAgentCalls)),
			zap.Int("delegateToolCalls", len(result.DelegateToolCalls)),
			zap.Int("regularToolCount", len(result.ToolCalls)),
		)

		// 构建完整的 tool_calls 列表（delegate + read-only），符合 OpenAI 对话格式
		allToolCalls := make([]llm.ToolCall, 0, len(result.DelegateToolCalls)+len(result.ToolCalls))
		allToolCalls = append(allToolCalls, result.DelegateToolCalls...)
		allToolCalls = append(allToolCalls, result.ToolCalls...)

		// 添加 assistant 消息（包含 tool_calls），这是 OpenAI function calling 必需的格式
		l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, allToolCalls))

		// 执行委托任务
		delegationResults := l.executeDelegations(ctx, result.SubAgentCalls)

		// 为每个 delegate_task 调用添加对应的 tool result 消息
		for _, dtc := range result.DelegateToolCalls {
			agentName, _, _ := tool.ExtractDelegation(dtc.Arguments)
			var content string
			if res, ok := delegationResults[agentName]; ok {
				if res.Success {
					content = res.Content
				} else {
					content = "错误: " + res.Error
				}
			} else {
				content = "错误: 未找到Agent执行结果"
			}
			l.state.History = append(l.state.History, llm.NewToolMessage(content, dtc.ID))
		}

		// 执行只读工具调用（如果有）
		if len(result.ToolCalls) > 0 {
			toolResults := l.executeTools(ctx, result.ToolCalls)
			for _, tr := range toolResults {
				l.state.History = append(l.state.History, llm.NewToolMessage(tr.Content, tr.ToolCallID))
			}
		}

		// 将委托结果存入上下文（供 Synthesize 阶段使用）
		if l.state.agentContext != nil {
			for _, res := range delegationResults {
				l.state.agentContext.AddAgentResult(res)
			}
		}

		// 同步 agentContext.History，确保下一轮 LLM 能看到完整历史
		if l.state.agentContext != nil {
			l.state.agentContext.History = l.state.History
		}

		return PhaseThink // 主Agent继续思考

	case agent.ActionSynthesize:
		log.Debug("Action: Synthesize")
		return PhaseSynthesize

	case agent.ActionWaitForInput:
		log.Debug("Action: WaitForInput")
		return PhaseWait
	case agent.ActionEndGame:
		log.Debug("Action: EndGame")
		return PhaseEnd
	case agent.ActionRespondToPlayer:
		log.Debug("Action: RespondToPlayer",
			zap.String("content", truncateString(result.Content, 200)),
		)
		return PhaseWait
	default:
		log.Debug("Action: default (WaitForInput)")
		return PhaseWait
	}
}

// executeDelegations 执行委托任务（调用SubAgent）
// 根据依赖分析决定串行或并行执行
func (l *ReActLoop) executeDelegations(ctx context.Context, calls []agent.SubAgentCall) map[string]*agent.AgentCallResult {
	if len(calls) == 0 {
		return nil
	}

	// 分析依赖关系，决定执行模式
	execMode := l.analyzeDependencies(calls)
	l.getLogger().Debug("Delegation execution mode",
		zap.String("mode", string(execMode)),
		zap.Int("callCount", len(calls)),
	)

	if execMode == agent.ExecutionParallel && len(calls) > 1 {
		return l.executeDelegationsParallel(ctx, calls)
	}
	return l.executeDelegationsSequential(ctx, calls)
}

// analyzeDependencies 分析委托任务的依赖关系
func (l *ReActLoop) analyzeDependencies(calls []agent.SubAgentCall) agent.ExecutionMode {
	// 如果只有一个委托，无需并行
	if len(calls) <= 1 {
		return agent.ExecutionSequential
	}

	// 收集所有目标Agent
	agentSet := make(map[string]bool)
	for _, call := range calls {
		agentSet[call.AgentName] = true
	}

	// 检查依赖关系
	// CombatAgent 依赖 CharacterAgent
	// 同一Agent的多个委托需要串行（状态冲突风险）
	for _, call := range calls {
		subAgent, ok := l.agents[call.AgentName]
		if !ok {
			continue
		}
		// 检查是否有依赖
		deps := subAgent.Dependencies()
		for _, dep := range deps {
			if agentSet[dep] {
				// 存在依赖，需要串行执行
				return agent.ExecutionSequential
			}
		}
	}

	// 检查是否有重复的Agent调用（同一Agent多次调用需串行）
	agentCount := make(map[string]int)
	for _, call := range calls {
		agentCount[call.AgentName]++
		if agentCount[call.AgentName] > 1 {
			return agent.ExecutionSequential
		}
	}

	// 无依赖且无重复，可以并行
	return agent.ExecutionParallel
}

// executeDelegationsSequential 串行执行委托任务
func (l *ReActLoop) executeDelegationsSequential(ctx context.Context, calls []agent.SubAgentCall) map[string]*agent.AgentCallResult {
	results := make(map[string]*agent.AgentCallResult)

	for _, call := range calls {
		result := l.executeSingleDelegation(ctx, call)
		results[call.AgentName] = result
	}

	return results
}

// executeDelegationsParallel 并行执行委托任务
func (l *ReActLoop) executeDelegationsParallel(ctx context.Context, calls []agent.SubAgentCall) map[string]*agent.AgentCallResult {
	results := make(map[string]*agent.AgentCallResult, len(calls))
	var mu sync.Mutex

	eg, ctx := errgroup.WithContext(ctx)

	for _, call := range calls {
		call := call // 闭包捕获
		eg.Go(func() error {
			result := l.executeSingleDelegation(ctx, call)
			mu.Lock()
			results[call.AgentName] = result
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		l.getLogger().Error("Parallel delegation execution error", zap.Error(err))
	}

	return results
}

// subAgentMaxIterations 子Agent内部循环最大迭代次数
const subAgentMaxIterations = 10

// executeSingleDelegation 执行单个委托任务
// 内部实现迷你ReAct循环：子Agent可能多次调用工具，直到生成最终文本响应
func (l *ReActLoop) executeSingleDelegation(ctx context.Context, call agent.SubAgentCall) *agent.AgentCallResult {
	log := l.getLogger()

	subAgent, ok := l.agents[call.AgentName]
	if !ok {
		return &agent.AgentCallResult{
			AgentName: call.AgentName,
			Success:   false,
			Error:     "未找到子Agent: " + call.AgentName,
		}
	}

	// 创建隔离的子会话上下文
	subCtx := l.createSubSession(l.state.agentContext)

	// 构建子Agent请求
	req := &agent.AgentRequest{
		UserInput: call.Intent,
		Context:   subCtx,
	}

	// 迷你ReAct循环：反复调用子Agent直到它生成最终文本响应
	for iteration := 0; iteration < subAgentMaxIterations; iteration++ {
		log.Debug("SubAgent iteration",
			zap.String("agentName", call.AgentName),
			zap.Int("iteration", iteration),
			zap.Int("subHistoryLength", len(subCtx.History)),
		)

		// 执行子Agent
		resp, err := subAgent.Execute(ctx, req)
		if err != nil {
			return &agent.AgentCallResult{
				AgentName: call.AgentName,
				Success:   false,
				Error:     "子Agent执行失败: " + err.Error(),
			}
		}

		// 如果没有Tool调用，说明子Agent已生成最终响应
		if len(resp.ToolCalls) == 0 {
			log.Debug("SubAgent completed with final response",
				zap.String("agentName", call.AgentName),
				zap.Int("iterations", iteration+1),
				zap.String("content", truncateString(resp.Content, 200)),
			)
			return &agent.AgentCallResult{
				AgentName: call.AgentName,
				Success:   true,
				Content:   resp.Content,
			}
		}

		// 有Tool调用：先将assistant消息（含tool_calls）加入子会话历史
		subCtx.History = append(subCtx.History, llm.NewAssistantMessage(resp.Content, resp.ToolCalls))

		// 执行工具并将结果加入子会话历史
		toolResults := l.tools.ExecuteTools(ctx, resp.ToolCalls)
		for _, tr := range toolResults {
			subCtx.History = append(subCtx.History, llm.NewToolMessage(tr.Content, tr.ToolCallID))
			log.Debug("SubAgent tool result",
				zap.String("agentName", call.AgentName),
				zap.String("toolCallID", tr.ToolCallID),
				zap.Bool("isError", tr.IsError),
				zap.String("content", truncateString(tr.Content, 100)),
			)
		}

		// 更新请求上下文，下一轮循环子Agent将看到工具结果
		// UserInput 只在第一轮传入，后续轮次清空避免重复
		req = &agent.AgentRequest{
			UserInput: "",
			Context:   subCtx,
		}
	}

	// 达到最大迭代次数，返回当前已有的内容
	log.Warn("SubAgent reached max iterations",
		zap.String("agentName", call.AgentName),
		zap.Int("maxIterations", subAgentMaxIterations),
	)

	// 从子会话历史中提取最后一次的内容作为结果
	var lastContent string
	for i := len(subCtx.History) - 1; i >= 0; i-- {
		if subCtx.History[i].Role == llm.RoleAssistant && subCtx.History[i].Content != "" {
			lastContent = subCtx.History[i].Content
			break
		}
	}

	return &agent.AgentCallResult{
		AgentName: call.AgentName,
		Success:   true,
		Content:   lastContent,
	}
}

// createSubSession 创建隔离的子会话上下文
func (l *ReActLoop) createSubSession(parentCtx *agent.AgentContext) *agent.AgentContext {
	if parentCtx == nil {
		return agent.NewAgentContext("", "", l.engine)
	}
	return &agent.AgentContext{
		GameID:       parentCtx.GameID,
		PlayerID:     parentCtx.PlayerID,
		Engine:       parentCtx.Engine,
		History:      make([]llm.Message, 0), // 独立历史
		CurrentState: parentCtx.CurrentState, // 共享游戏状态（只读）
		Metadata:     make(map[string]any),
		Parent:       parentCtx, // 链接父会话
		IsSubSession: true,
	}
}

// joinStrings 连接字符串切片
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
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

// synthesize 合成阶段 - 将多个Agent的执行结果整合为连贯叙事
func (l *ReActLoop) synthesize(ctx context.Context) {
	log := l.getLogger()
	log.Debug("Synthesize phase started")

	// 检查是否有Agent结果需要合成
	if l.state.agentContext == nil || !l.state.agentContext.HasAgentResults() {
		log.Debug("No agent results to synthesize, transitioning to Think phase")
		l.state.CurrentPhase = PhaseThink
		return
	}

	// 获取用户输入
	var userInput string
	if len(l.state.History) > 0 {
		lastMsg := l.state.History[len(l.state.History)-1]
		if lastMsg.Role == llm.RoleUser {
			userInput = lastMsg.Content
		}
	}

	// 构建Agent结果摘要
	var resultSummaries []string
	agentResults := l.state.agentContext.GetAllAgentResults()
	for _, res := range agentResults {
		if res.Success {
			resultSummaries = append(resultSummaries, fmt.Sprintf("[%s] %s", res.AgentName, res.Content))
		} else {
			resultSummaries = append(resultSummaries, fmt.Sprintf("[%s] 错误: %s", res.AgentName, res.Error))
		}
	}

	// 构建合成请求
	synthesisPrompt := l.buildSynthesisPrompt(userInput, joinStrings(resultSummaries, "\n"))

	// 调用LLM进行合成
	messages := []llm.Message{
		llm.NewSystemMessage(synthesisPrompt),
		llm.NewUserMessage("请合成以上Agent的执行结果，输出流畅的叙事描述。"),
	}

	req := &llm.CompletionRequest{
		Messages: messages,
	}
	resp, err := l.llm.Complete(ctx, req)
	if err != nil {
		log.Error("Synthesis LLM call failed", zap.Error(err))
		// 失败时使用简单拼接
		l.state.History = append(l.state.History, llm.NewAssistantMessage(
			fmt.Sprintf("委托任务执行完成:\n%s", joinStrings(resultSummaries, "\n")),
			nil,
		))
	} else {
		// 将合成结果添加到历史
		l.state.History = append(l.state.History, llm.NewAssistantMessage(resp.Content, nil))
		log.Debug("Synthesis completed",
			zap.String("content", truncateString(resp.Content, 200)),
		)
	}

	// 清空Agent结果，准备下一轮
	l.state.agentContext.ClearAgentResults()

	// 转入等待阶段
	l.state.CurrentPhase = PhaseWait
}

// buildSynthesisPrompt 构建合成提示词
func (l *ReActLoop) buildSynthesisPrompt(userInput, agentResults string) string {
	// 使用简化的合成提示词
	return fmt.Sprintf(`你是一个结果合成专家，负责将多个专业Agent的执行结果整合为一个连贯的叙事输出。

## 输入信息

**玩家请求**：
%s

**Agent执行结果**：
%s

## 输出要求

1. 将多个Agent的输出整合为流畅的叙事，避免机械拼接
2. 保持D&D游戏风格，以DM（地下城主）的口吻描述
3. 强调重要的游戏事件（伤害、技能检定、状态变化）
4. 保留关键数值的准确性
5. 输出一段完整的叙事段落

请直接输出合成后的叙事，不要添加额外说明：`, userInput, agentResults)
}
