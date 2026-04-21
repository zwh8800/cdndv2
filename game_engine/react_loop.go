package gameengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

// maxDelegationsPerRun 每次 ReAct 循环最大委托次数
const maxDelegationsPerRun = 15

// LoopState 循环状态
type LoopState struct {
	GameID          model.ID
	PlayerID        model.ID
	History         []llm.Message
	CurrentPhase    Phase
	Iteration       int
	LastResult      *agent.AgentResponse
	agentContext    *agent.AgentContext
	delegationCount int
	actionHistory   []ActionRecord
}

// ActionRecord 记录每次工具调用的名称、参数和结果摘要，用于幻觉循环检测
type ActionRecord struct {
	ToolName     string
	Params       map[string]any
	ResultDigest string
}

// ReActLoop ReAct循环控制器
type ReActLoop struct {
	engine     *engine.Engine
	mainAgent  agent.Agent
	router     *agent.RouterAgent
	agents     map[string]agent.SubAgent
	tools      *tool.ToolRegistry
	llm        llm.LLMClient
	state      *LoopState
	maxIter    int
	useRouter  bool
	logger     *zap.Logger
	compressor *llm.ContextCompressor // 上下文压缩器
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
	loop := &ReActLoop{
		engine:     e,
		mainAgent:  mainAgent,
		router:     router,
		agents:     agents,
		tools:      tools,
		llm:        llmClient,
		maxIter:    maxIter,
		useRouter:  router != nil,
		compressor: llm.DefaultContextCompressor(llmClient),
		state: &LoopState{
			CurrentPhase: PhaseObserve,
			History:      make([]llm.Message, 0),
		},
		logger: zap.NewNop(),
	}

	loop.compressor.SetToolReadOnlyChecker(tools)

	return loop
}

// SetLogger 设置日志器
func (l *ReActLoop) SetLogger(log *zap.Logger) {
	if log != nil {
		l.logger = log
	}
}

// SetCompressor 设置上下文压缩器
func (l *ReActLoop) SetCompressor(compressor *llm.ContextCompressor) {
	if compressor != nil {
		l.compressor = compressor
	}
}

// maybeCompressHistory 检查并在需要时触发异步压缩
// 两步式流程：
// Step 1: 检查是否有已完成的后台压缩结果，如有则应用
// Step 2: 检查是否需要启动新的后台压缩（不阻塞当前请求）
func (l *ReActLoop) maybeCompressHistory() {
	log := l.getLogger()

	if l.compressor == nil {
		return
	}

	// Step 1: 检查是否有已完成的后台压缩结果，如有则应用
	if compressed := l.compressor.ApplyCompressedIfReady(); compressed != nil {
		beforeLen := len(l.state.History)
		l.state.History = compressed
		log.Info("Applied async compressed history",
			zap.Int("beforeMessages", beforeLen),
			zap.Int("afterMessages", len(compressed)),
		)
		// 同步到 agentContext
		if l.state.agentContext != nil {
			l.state.agentContext.History = l.state.History
		}
		return
	}

	// Step 2: 检查是否需要启动新的后台压缩
	// 如果已有压缩任务在运行，跳过本次检查，避免重复启动
	if l.compressor.IsCompressing() {
		return
	}
	if l.compressor.NeedsCompression(l.state.History) {
		// 拷贝当前历史快照，启动后台压缩
		snapshot := make([]llm.Message, len(l.state.History))
		copy(snapshot, l.state.History)
		l.compressor.StartAsyncCompress(context.Background(), snapshot)
		log.Info("Async compression started in background",
			zap.Int("historyMessages", len(snapshot)),
			zap.Int("estimatedTokens", l.compressor.EstimateTokens(snapshot)),
		)
		// 当前请求继续使用未压缩的完整历史，不阻塞
	}
}

// Run 执行ReAct循环
func (l *ReActLoop) Run(ctx context.Context, initialInput string, gameID, playerID model.ID) error {
	log := l.getLogger()

	l.state.GameID = gameID
	l.state.PlayerID = playerID
	l.state.Iteration = 0
	l.state.CurrentPhase = PhaseObserve
	l.state.delegationCount = 0

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

	// 自动推进游戏阶段：当处于 character_creation 且已存在玩家角色时，自动进入 exploration
	if summary != nil && summary.Phase == "character_creation" && summary.Player != nil {
		log.Info("Auto-advancing game phase from character_creation to exploration",
			zap.String("playerName", summary.Player.Name),
			zap.String("playerID", string(summary.Player.ID)),
		)
		_, err := l.engine.SetPhase(ctx, l.state.GameID, model.PhaseExploration, "角色创建完成，自动进入探索阶段")
		if err != nil {
			log.Warn("Failed to auto-advance phase", zap.Error(err))
		} else {
			summary.Phase = "exploration"
		}
	}

	// 构建上下文
	l.state.agentContext = agent.NewAgentContext(
		string(l.state.GameID),
		string(l.state.PlayerID),
		l.engine,
	)
	l.state.agentContext.History = l.state.History
	l.state.agentContext.CurrentState = summary

	// 从游戏状态摘要中预填充已知实体ID
	if summary != nil {
		l.populateKnownEntityIDsFromSummary(summary)
	}

	// 检查并执行上下文压缩（在进入 Think/Route 阶段前）
	l.maybeCompressHistory()

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
		l.state.delegationCount++
		if l.state.delegationCount > maxDelegationsPerRun {
			log.Warn("Max delegations reached in route phase, skipping further delegations",
				zap.Int("delegationCount", l.state.delegationCount),
			)
			l.state.CurrentPhase = PhaseThink
			return
		}

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

	// 校准 token 估算（利用 LLM 返回的实际 usage）
	if l.compressor != nil && resp.Usage.PromptTokens > 0 {
		estimated := l.compressor.EstimateTokens(l.state.History)
		l.compressor.CalibrateWithActualUsage(estimated, resp.Usage.PromptTokens)
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

		// 幻觉循环检测：记录 delegate_task 调用
		for _, dtc := range result.DelegateToolCalls {
			l.recordAction(dtc.Name, dtc.Arguments, nil)
		}
		for _, tc := range result.ToolCalls {
			l.recordAction(tc.Name, tc.Arguments, nil)
		}
		if l.detectHallucinationLoop() {
			log.Warn("Hallucination loop detected in delegate phase, injecting warning")
			l.state.History = append(l.state.History, llm.NewSystemMessage(
				"[系统警告] 检测到重复操作：你已连续多次委托相同任务。请尝试不同的方法或直接回复玩家。",
			))
		}

		l.state.delegationCount++
		if l.state.delegationCount >= maxDelegationsPerRun {
			log.Debug("Max delegations reached, forcing response",
				zap.Int("delegationCount", l.state.delegationCount),
				zap.Int("maxDelegations", maxDelegationsPerRun),
			)
			return PhaseSynthesize
		}

		// 检查压缩（委托完成后可能追加了大量 tool 消息）
		l.maybeCompressHistory()
		return PhaseThink

	case agent.ActionContinue:
		// 纯只读工具调用（无委托）
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

			l.state.History = append(l.state.History, llm.NewAssistantMessage("", result.ToolCalls))
			log.Debug("Assistant tool call message added to history",
				zap.Int("toolCallCount", len(result.ToolCalls)),
			)

			toolResults := l.executeTools(ctx, result.ToolCalls)

			for _, tr := range toolResults {
				l.state.History = append(l.state.History, llm.NewToolMessage(tr.Content, tr.ToolCallID))
				log.Debug("Tool result added to history",
					zap.String("toolCallID", tr.ToolCallID),
					zap.Bool("isError", tr.IsError),
					zap.String("content", truncateString(tr.Content, 100)),
				)
			}

			if l.state.agentContext != nil {
				l.state.agentContext.History = l.state.History
			}

			// 幻觉循环检测：记录工具调用并检测重复模式
			for _, tc := range result.ToolCalls {
				l.recordAction(tc.Name, tc.Arguments, toolResults)
			}
			if l.detectHallucinationLoop() {
				log.Warn("Hallucination loop detected: same tool called 3 times with similar results, injecting warning")
				l.state.History = append(l.state.History, llm.NewSystemMessage(
					"[系统警告] 检测到重复操作：你已连续多次执行相同操作并得到相同结果。请尝试不同的方法或直接回复玩家。",
				))
			}

			// 检查压缩（委托完成后可能追加了大量 tool 消息）
			l.maybeCompressHistory()
			return PhaseThink
		}

		// 无工具调用但有内容，视为响应
		if result.Content != "" {
			l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, nil))
			log.Debug("Assistant message added to history",
				zap.String("content", truncateString(result.Content, 200)),
			)
		}
		return PhaseWait

	case agent.ActionSynthesize:
		log.Debug("Action: Synthesize")
		return PhaseSynthesize

	case agent.ActionWaitForInput:
		log.Debug("Action: WaitForInput")
		if result.Content != "" {
			l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, nil))
		}
		return PhaseWait
	case agent.ActionEndGame:
		log.Debug("Action: EndGame")
		if result.Content != "" {
			l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, nil))
		}
		return PhaseEnd
	case agent.ActionRespondToPlayer:
		log.Debug("Action: RespondToPlayer",
			zap.String("content", truncateString(result.Content, 200)),
		)
		if result.Content != "" {
			l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, nil))
		}
		return PhaseWait
	default:
		log.Debug("Action: default (WaitForInput)")
		if result.Content != "" {
			l.state.History = append(l.state.History, llm.NewAssistantMessage(result.Content, nil))
		}
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

		// 从执行结果中提取实体ID并注入到上下文中，供后续 SubAgent 使用
		if result.Success && l.state.agentContext != nil {
			entityIDs := extractEntityIDsFromResult(result)
			if len(entityIDs) > 0 {
				l.state.agentContext.MergeKnownEntityIDs(entityIDs)
			}
		}
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
const subAgentMaxIterations = 5

// CombatPlan 战斗计划（Plan-Then-Act 模式）
type CombatPlan struct {
	PlanType     string          `json:"plan_type"` // full_round, partial, single_action
	Actions      []CombatPlanAction `json:"actions"`
	Contingency string          `json:"contingency,omitempty"`
}

// CombatPlanAction 战斗计划中的单个动作
type CombatPlanAction struct {
	Tool   string         `json:"tool"`
	Params map[string]any `json:"params"`
	Reason string         `json:"reason,omitempty"`
}

// executeSingleDelegation 执行单个委托任务
// 内部实现迷你ReAct循环：子Agent可能多次调用工具，直到生成最终文本响应
// 对于CombatAgent支持Plan-Then-Act：先生成完整计划，再自动依次执行
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

		// 如果没有Tool调用，检查是否是Plan-Then-Act战斗计划
		if len(resp.ToolCalls) == 0 && call.AgentName == agent.SubAgentNameCombat {
			// 尝试解析为战斗计划
			jsonContent := extractJSONFromString(resp.Content)
			if jsonContent != "" {
				var plan CombatPlan
				if err := json.Unmarshal([]byte(jsonContent), &plan); err == nil && len(plan.Actions) > 0 {
					log.Debug("Combat plan detected, executing Plan-Then-Act",
						zap.Int("actionCount", len(plan.Actions)),
						zap.String("planType", plan.PlanType),
					)

					// 按顺序执行计划中的每个动作
					executionResults := make([]string, 0)
					allSucceeded := true

					for i, action := range plan.Actions {
						log.Debug("Executing planned action",
							zap.Int("index", i),
							zap.String("tool", action.Tool),
						)

						// 创建tool call
						tc := llm.ToolCall{
							Name:      action.Tool,
							Arguments: action.Params,
							ID:        fmt.Sprintf("plan_%d_%s", i, action.Tool),
						}

						// 解析名称为ID并执行
						l.resolveToolNames(ctx, []llm.ToolCall{tc})
						results := l.tools.ExecuteTools(ctx, []llm.ToolCall{tc})

						// 添加到子会话历史
						subCtx.History = append(subCtx.History, llm.NewAssistantMessage("", []llm.ToolCall{tc}))
						for _, tr := range results {
							subCtx.History = append(subCtx.History, llm.NewToolMessage(tr.Content, tr.ToolCallID))
							executionResults = append(executionResults, fmt.Sprintf("[%s] %s", action.Tool, tr.Content))
							if tr.IsError {
								allSucceeded = false
								executionResults = append(executionResults, fmt.Sprintf("[%s] 错误: %s", action.Tool, tr.Content))
							}
						}
					}

					// 构建执行结果总结
					var summary string
					if allSucceeded {
						summary = fmt.Sprintf("战斗计划执行完成 (%d 个动作全部成功)。\n\n", len(plan.Actions))
					} else {
						summary = fmt.Sprintf("战斗计划执行完成（部分动作失败）。\n\n")
					}
					if plan.Contingency != "" {
						summary += fmt.Sprintf("应急计划: %s\n\n", plan.Contingency)
					}
					summary += "执行结果:\n" + strings.Join(executionResults, "\n")

					log.Debug("Combat plan execution completed",
						zap.Bool("allSucceeded", allSucceeded),
					)

					return &agent.AgentCallResult{
						AgentName: call.AgentName,
						Success:   true,
						Content:   summary,
					}
				}
			}

			// 不是计划，就是普通的最终响应
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

		// 如果没有Tool调用，就是普通的最终响应
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

		// 解析名称为ID并执行工具
		l.resolveToolNames(ctx, resp.ToolCalls)
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

// subSessionParentContextMessages 从父会话历史中提取的上下文消息数量上限
const subSessionParentContextMessages = 20

// createSubSession 创建隔离的子会话上下文
// 从父会话历史中提取最近的有意义对话作为上下文，让 SubAgent 了解之前发生了什么
func (l *ReActLoop) createSubSession(parentCtx *agent.AgentContext) *agent.AgentContext {
	if parentCtx == nil {
		return agent.NewAgentContext("", "", l.engine)
	}
	subCtx := &agent.AgentContext{
		GameID:       parentCtx.GameID,
		PlayerID:     parentCtx.PlayerID,
		Engine:       parentCtx.Engine,
		History:      l.extractParentContext(parentCtx.History), // 从父历史提取上下文
		CurrentState: parentCtx.CurrentState,                    // 共享游戏状态（只读）
		Metadata:     make(map[string]any),
		Parent:       parentCtx, // 链接父会话
		IsSubSession: true,
	}
	// 继承父会话的已知实体ID，确保 SubAgent 间可以共享 actor_id、scene_id 等
	if parentCtx.KnownEntityIDs != nil {
		subCtx.KnownEntityIDs = make(map[string]string)
		for k, v := range parentCtx.KnownEntityIDs {
			subCtx.KnownEntityIDs[k] = v
		}
	}
	return subCtx
}

// extractParentContext 从父会话历史中提取最近的有意义对话消息
// 只保留 user 和 assistant（有内容且无tool_calls）消息，过滤掉 tool/system 消息
// 这些消息以 "对话上下文" 的形式传递给 SubAgent，帮助其理解当前情境
func (l *ReActLoop) extractParentContext(parentHistory []llm.Message) []llm.Message {
	if len(parentHistory) == 0 {
		return make([]llm.Message, 0)
	}

	// 从后往前扫描，收集有意义的消息（user 和有内容的 assistant）
	var relevant []llm.Message
	for i := len(parentHistory) - 1; i >= 0 && len(relevant) < subSessionParentContextMessages; i-- {
		msg := parentHistory[i]
		switch msg.Role {
		case llm.RoleUser:
			if msg.Content != "" {
				relevant = append(relevant, msg)
			}
		case llm.RoleAssistant:
			// 只保留有实际内容的 assistant 消息（跳过纯 tool_calls 的）
			if msg.Content != "" && len(msg.ToolCalls) == 0 {
				relevant = append(relevant, msg)
			}
		}
	}

	if len(relevant) == 0 {
		return make([]llm.Message, 0)
	}

	// 反转为时间顺序
	for i, j := 0, len(relevant)-1; i < j; i, j = i+1, j-1 {
		relevant[i], relevant[j] = relevant[j], relevant[i]
	}

	// 用一条 user 消息包装为上下文摘要，避免 SubAgent 把这些当成自己的对话历史
	var contextParts []string
	for _, msg := range relevant {
		switch msg.Role {
		case llm.RoleUser:
			contextParts = append(contextParts, "玩家: "+msg.Content)
		case llm.RoleAssistant:
			contextParts = append(contextParts, "DM: "+msg.Content)
		}
	}

	contextMsg := llm.NewUserMessage(
		"[最近对话上下文 - 以下是主会话中最近的对话记录，请基于此理解当前情境]\n" +
			joinStrings(contextParts, "\n"),
	)

	return []llm.Message{contextMsg}
}

// populateKnownEntityIDsFromSummary 从游戏状态摘要中提取已知实体ID
func (l *ReActLoop) populateKnownEntityIDsFromSummary(summary *game_summary.GameSummary) {
	if summary == nil || l.state.agentContext == nil {
		return
	}

	// 提取场景ID
	if summary.CurrentScene != nil && summary.CurrentScene.ID != "" {
		l.state.agentContext.SetKnownEntityID("scene_id", string(summary.CurrentScene.ID))
	}

	// 提取玩家角色ID
	if summary.Player != nil && summary.Player.ID != "" {
		l.state.agentContext.SetKnownEntityID("actor_id", string(summary.Player.ID))
	}
}

// extractEntityIDsFromResult 从 SubAgent 执行结果中提取实体ID
// 解析 Content 中的 JSON 数据，查找 actor_id、scene_id 等关键字段
func extractEntityIDsFromResult(result *agent.AgentCallResult) map[string]string {
	entityIDs := make(map[string]string)

	// 从 Content 中尝试提取 JSON
	jsonContent := extractJSONFromString(result.Content)
	if jsonContent == "" {
		return entityIDs
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		return entityIDs
	}

	// 提取 actor_id
	if id, ok := findFieldInJSON(data, "actor_id"); ok {
		entityIDs["actor_id"] = id
	}
	// 提取 scene_id
	if id, ok := findFieldInJSON(data, "scene_id"); ok {
		entityIDs["scene_id"] = id
	}
	// 提取 actor_id 嵌套在 scene.actors 或 actor 中
	if scene, ok := data["scene"].(map[string]any); ok {
		if id, ok := scene["id"].(string); ok && id != "" {
			entityIDs["scene_id"] = id
		}
	}

	return entityIDs
}

// extractJSONFromString 从字符串中提取 JSON 对象
func extractJSONFromString(s string) string {
	start := -1
	for i, c := range s {
		if c == '{' {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// findFieldInJSON 在 JSON 数据中递归查找指定字段
func findFieldInJSON(data map[string]any, field string) (string, bool) {
	if val, ok := data[field].(string); ok && val != "" {
		return val, true
	}
	// 递归查找嵌套对象
	for _, v := range data {
		if nested, ok := v.(map[string]any); ok {
			if val, ok := findFieldInJSON(nested, field); ok {
				return val, true
			}
		}
	}
	return "", false
}

// formatKnownEntityIDs 格式化已知实体ID为可读文本，注入到 SubAgent system prompt
func formatKnownEntityIDs(entityIDs map[string]string) string {
	if len(entityIDs) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "## 已知实体ID（重要：调用 API 时必须使用以下 ID）")
	parts = append(parts, "")

	if actorID, ok := entityIDs["actor_id"]; ok {
		parts = append(parts, fmt.Sprintf("- **角色ID (actor_id)**: `%s`", actorID))
	}
	if sceneID, ok := entityIDs["scene_id"]; ok {
		parts = append(parts, fmt.Sprintf("- **场景ID (scene_id)**: `%s`", sceneID))
	}

	parts = append(parts, "")
	parts = append(parts, "**注意**: 在调用任何需要 actor_id 或 scene_id 的 API 时，必须使用上述 ID 值，不得使用角色名称或其他标识符。")

	return joinStrings(parts, "\n")
}

// recordAction 记录一次工具调用到动作历史，用于幻觉循环检测
func (l *ReActLoop) recordAction(toolName string, params map[string]any, results []llm.ToolResult) {
	digest := ""
	if len(results) > 0 {
		r := results[0]
		if len(r.Content) > 100 {
			digest = r.Content[:100]
		} else {
			digest = r.Content
		}
	}
	l.state.actionHistory = append(l.state.actionHistory, ActionRecord{
		ToolName:     toolName,
		Params:       params,
		ResultDigest: digest,
	})

	// 保留最近10条记录，防止无限增长
	if len(l.state.actionHistory) > 10 {
		l.state.actionHistory = l.state.actionHistory[len(l.state.actionHistory)-10:]
	}
}

// detectHallucinationLoop 检测幻觉循环：连续3次相同工具+相似参数+相似结果
func (l *ReActLoop) detectHallucinationLoop() bool {
	history := l.state.actionHistory
	if len(history) < 3 {
		return false
	}

	recent := history[len(history)-3:]

	sameTool := recent[0].ToolName == recent[1].ToolName && recent[1].ToolName == recent[2].ToolName
	if !sameTool {
		return false
	}

	similarParams := similarMaps(recent[0].Params, recent[1].Params) && similarMaps(recent[1].Params, recent[2].Params)
	similarResults := recent[0].ResultDigest == recent[1].ResultDigest && recent[1].ResultDigest == recent[2].ResultDigest

	return similarParams && similarResults
}

// similarMaps 粗略比较两个 map 是否相似（比较大小和主要键值）
func similarMaps(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	matches := 0
	for k, va := range a {
		if vb, ok := b[k]; ok {
			if fmt.Sprintf("%v", va) == fmt.Sprintf("%v", vb) {
				matches++
			}
		}
	}
	return matches >= len(a)-1
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

// executeTools 执行Tools（带名称解析）
func (l *ReActLoop) executeTools(ctx context.Context, calls []llm.ToolCall) []llm.ToolResult {
	// 在执行前解析工具参数中的名称为ID
	l.resolveToolNames(ctx, calls)
	return l.tools.ExecuteTools(ctx, calls)
}

// resolveToolNames 解析工具调用参数中的名称为实体ID
// 当 LLM 使用角色名或场景名而非 ID 时，自动查找并替换为正确的 ID
func (l *ReActLoop) resolveToolNames(ctx context.Context, calls []llm.ToolCall) {
	log := l.getLogger()
	for _, call := range calls {
		l.resolveSingleCallNames(ctx, call)
	}
	_ = log
}

// resolveSingleCallNames 解析单个工具调用的名称参数
func (l *ReActLoop) resolveSingleCallNames(ctx context.Context, call llm.ToolCall) {
	if call.Arguments == nil {
		return
	}

	// 解析 actor_id
	if val, ok := call.Arguments["actor_id"].(string); ok && val != "" {
		if resolved := l.resolveActorName(ctx, val); resolved != "" {
			call.Arguments["actor_id"] = resolved
		}
	}
	// 解析 caster_id
	if val, ok := call.Arguments["caster_id"].(string); ok && val != "" {
		if resolved := l.resolveActorName(ctx, val); resolved != "" {
			call.Arguments["caster_id"] = resolved
		}
	}
	// 解析 pc_id
	if val, ok := call.Arguments["pc_id"].(string); ok && val != "" {
		if resolved := l.resolveActorName(ctx, val); resolved != "" {
			call.Arguments["pc_id"] = resolved
		}
	}
	// 解析 target_actor_id
	if val, ok := call.Arguments["target_actor_id"].(string); ok && val != "" {
		if resolved := l.resolveActorName(ctx, val); resolved != "" {
			call.Arguments["target_actor_id"] = resolved
		}
	}
	// 解析 scene_id
	if val, ok := call.Arguments["scene_id"].(string); ok && val != "" {
		if resolved := l.resolveSceneName(ctx, val); resolved != "" {
			call.Arguments["scene_id"] = resolved
		}
	}
}

// resolveActorName 将角色名称解析为 actor_id
func (l *ReActLoop) resolveActorName(ctx context.Context, nameOrID string) string {
	// 如果已经是 ID 格式，直接返回
	if strings.HasPrefix(nameOrID, "01") || strings.HasPrefix(nameOrID, "actor_") {
		return nameOrID
	}

	// 先从 KnownEntityIDs 中查找（按名称映射）
	if l.state.agentContext != nil {
		for key, id := range l.state.agentContext.KnownEntityIDs {
			if strings.HasSuffix(key, "_name") && strings.EqualFold(l.state.agentContext.KnownEntityIDs[key], nameOrID) {
				return id
			}
		}
	}

	// 通过引擎查询
	result, err := l.engine.ListActors(ctx, engine.ListActorsRequest{
		GameID: l.state.GameID,
	})
	if err != nil || result == nil {
		return nameOrID
	}

	// 按名称匹配
	for _, actor := range result.Actors {
		if strings.EqualFold(actor.Name, nameOrID) {
			return string(actor.ID)
		}
	}

	// 未找到，返回原始值（让工具自己处理错误）
	return nameOrID
}

// resolveSceneName 将场景名称解析为 scene_id
func (l *ReActLoop) resolveSceneName(ctx context.Context, nameOrID string) string {
	if strings.HasPrefix(nameOrID, "01") || strings.HasPrefix(nameOrID, "scene_") {
		return nameOrID
	}

	result, err := l.engine.ListScenes(ctx, engine.ListScenesRequest{
		GameID: l.state.GameID,
	})
	if err != nil || result == nil {
		return nameOrID
	}

	for _, scene := range result.Scenes {
		if strings.EqualFold(scene.Name, nameOrID) {
			return string(scene.ID)
		}
	}

	return nameOrID
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
