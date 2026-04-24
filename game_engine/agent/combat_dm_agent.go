package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/prompt"
	"github.com/zwh8800/cdndv2/game_engine/tool"
	dndengine "github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

const (
	maxReActIterations = 20  // 内层 ReAct 循环最大迭代次数
	maxEnemyTurnChain  = 10  // 外层循环连续敌人回合安全阀
	maxHistoryMessages = 100 // 历史消息上限，超过后截断
	keepRecentMessages = 30  // 截断时保留最近的消息条数
)

// CombatAgentResult CombatDMAgent 的运行结果
type CombatAgentResult struct {
	Response    string // 返回给玩家的叙述文本
	CombatEnded bool   // 战斗是否结束
	Summary     string // 战斗结束时交还给 MainAgent 的事实摘要
}

type combatIntentType string

const (
	combatIntentAction  combatIntentType = "action"
	combatIntentEndTurn combatIntentType = "end_turn"
	combatIntentInvalid combatIntentType = "invalid"
)

// ResolvedCombatIntent 是 LLM/启发式解析后的受控动作意图。
type ResolvedCombatIntent struct {
	IntentType combatIntentType `json:"intent_type"`
	ActionID   string           `json:"action_id,omitempty"`
	TargetID   model.ID         `json:"target_id,omitempty"`
	TargetIDs  []model.ID       `json:"target_ids,omitempty"`
	Confidence float64          `json:"confidence,omitempty"`
	Reason     string           `json:"reason,omitempty"`
}

type combatActionEvent struct {
	ActorID        model.ID
	ActorName      string
	ActorType      string
	OriginalInput  string
	Intent         ResolvedCombatIntent
	ExecuteResult  *dndengine.ExecuteTurnActionResult
	NextTurnResult *dndengine.NextTurnResult
	CombatEnded    bool
	Message        string
}

// turnStateCache 从 dnd-core 引擎查询的当前回合状态
type turnStateCache struct {
	ActorID   model.ID `json:"actor_id"`
	ActorName string   `json:"actor_name"`
	ActorType string   `json:"actor_type"` // "pc", "enemy", "npc", "companion"
}

// CombatDMAgent 自主战斗DM Agent
// 通过 LLM function calling 驱动战斗工具，管理完整战斗流程。
// 采用两层循环架构：
//   - 内层：LLM ReAct tool-calling 循环
//   - 外层：Go 框架层根据回合归属进行路由（yield 给玩家 / 调用 EnemyAI）
type CombatDMAgent struct {
	gameID    model.ID
	playerID  model.ID
	llmClient llm.LLMClient
	registry  *tool.ToolRegistry
	dndEngine *dndengine.Engine // dnd-core 引擎，用于直接查询游戏状态
	enemyAI   *EnemyAIAgent
	history   []llm.Message
	logger    *zap.Logger

	turnState     *turnStateCache                   // 当前回合状态（从引擎查询）
	cachedActions *dndengine.AvailableActionsResult // 缓存的当前角色可用动作
	ended         bool                              // 战斗结束标记
	events        []combatActionEvent               // 战斗事实事件，用于战后摘要
	summary       string                            // 已生成的战后摘要

	toolSchemas []map[string]any // 缓存的工具 schema（LLM 格式）
}

// NewCombatDMAgent 创建自主战斗DM Agent
func NewCombatDMAgent(
	gameID, playerID model.ID,
	llmClient llm.LLMClient,
	registry *tool.ToolRegistry,
	dndEngine *dndengine.Engine,
	enemyAI *EnemyAIAgent,
	logger *zap.Logger,
) *CombatDMAgent {
	if logger == nil {
		logger = zap.NewNop()
	}

	agent := &CombatDMAgent{
		gameID:    gameID,
		playerID:  playerID,
		llmClient: llmClient,
		registry:  registry,
		dndEngine: dndEngine,
		enemyAI:   enemyAI,
		history:   make([]llm.Message, 0),
		logger:    logger,
	}

	// 预构建工具 schema
	agent.toolSchemas = agent.buildToolSchemas()

	return agent
}

// Name 返回Agent名称
func (a *CombatDMAgent) Name() string {
	return CombatDMAgentName
}

// IsEnded 返回战斗是否已结束
func (a *CombatDMAgent) IsEnded() bool {
	return a.ended
}

// Summary 返回战斗结束时生成的事实摘要。
func (a *CombatDMAgent) Summary() string {
	return a.summary
}

// Initialize 初始化战斗会话
// 添加系统提示词到历史，然后运行第一轮循环（无玩家输入）
func (a *CombatDMAgent) Initialize(ctx context.Context) (*CombatAgentResult, error) {
	a.logger.Info("CombatDMAgent initializing",
		zap.String("gameID", string(a.gameID)),
	)

	a.history = append(a.history, llm.NewSystemMessage("你是D&D 5e战斗叙事助手。规则计算和回合推进由系统完成。"))

	return a.RunLoop(ctx, "")
}

// RunLoop 主运行循环。Go 层负责所有回合推进和 dnd-core 调用；
// LLM 只参与自然语言动作解析与叙事生成。
func (a *CombatDMAgent) RunLoop(ctx context.Context, input string) (*CombatAgentResult, error) {
	if a.ended {
		if a.summary == "" {
			a.summary = a.generateCombatSummary(ctx, nil)
		}
		return &CombatAgentResult{Response: "战斗已经结束。", CombatEnded: true, Summary: a.summary}, nil
	}
	if input != "" {
		a.history = append(a.history, llm.NewUserMessage(input))
	}

	var allResponses []string
	if err := a.refreshTurnState(ctx); err != nil {
		return nil, err
	}
	if a.ended {
		if a.summary == "" {
			a.summary = a.generateCombatSummary(ctx, nil)
		}
		return &CombatAgentResult{Response: "战斗已经结束。", CombatEnded: true, Summary: a.summary}, nil
	}

	if a.isPlayerTurn() {
		if input == "" {
			return &CombatAgentResult{
				Response:    a.describeCurrentTurn(ctx),
				CombatEnded: false,
			}, nil
		}

		event, err := a.processCurrentActorInput(ctx, input, true)
		if err != nil {
			return nil, err
		}
		if event.Message != "" {
			allResponses = append(allResponses, event.Message)
		}
		if event.CombatEnded {
			return &CombatAgentResult{Response: strings.Join(allResponses, "\n\n"), CombatEnded: true, Summary: a.summary}, nil
		}
		if err := a.refreshTurnState(ctx); err != nil {
			return nil, err
		}
		if a.isPlayerTurn() {
			return &CombatAgentResult{Response: strings.Join(allResponses, "\n\n"), CombatEnded: false}, nil
		}
	} else if input != "" {
		allResponses = append(allResponses, "当前还没有轮到你行动，我会先处理当前行动者的回合。")
	}

	enemyResponses, ended, err := a.runUntilPlayerTurnOrCombatEnd(ctx)
	if err != nil {
		return nil, err
	}
	allResponses = append(allResponses, enemyResponses...)
	if len(allResponses) == 0 {
		allResponses = append(allResponses, a.describeCurrentTurn(ctx))
	}
	return &CombatAgentResult{
		Response:    strings.Join(allResponses, "\n\n"),
		CombatEnded: ended,
		Summary:     a.summary,
	}, nil
}

func (a *CombatDMAgent) runUntilPlayerTurnOrCombatEnd(ctx context.Context) ([]string, bool, error) {
	var responses []string

	for chainCount := 0; chainCount < maxEnemyTurnChain; chainCount++ {
		if err := a.refreshTurnState(ctx); err != nil {
			return nil, false, err
		}
		if a.ended {
			return responses, true, nil
		}
		if a.isPlayerTurn() {
			if len(responses) == 0 {
				responses = append(responses, a.describeCurrentTurn(ctx))
			}
			return responses, false, nil
		}

		if !hasProactiveActions(a.cachedActions) {
			event, err := a.advanceTurn(ctx, "当前行动者没有可执行动作，自动结束回合。")
			if err != nil {
				return nil, false, err
			}
			if event.Message != "" {
				responses = append(responses, event.Message)
			}
			if event.CombatEnded {
				return responses, true, nil
			}
			continue
		}

		intentText, err := a.generateEnemyIntent(ctx)
		if err != nil {
			a.logger.Error("Failed to generate enemy intent", zap.Error(err))
			intentText = "采取最直接的攻击行动"
		}
		a.history = append(a.history, llm.NewUserMessage(fmt.Sprintf("[%s的行动] %s", a.turnState.ActorName, intentText)))

		event, err := a.processCurrentActorInput(ctx, intentText, false)
		if err != nil {
			return nil, false, err
		}
		if event.Message != "" {
			responses = append(responses, event.Message)
		}
		if event.CombatEnded {
			return responses, true, nil
		}
	}

	a.logger.Warn("Max enemy turn chain reached", zap.Int("chainCount", maxEnemyTurnChain))
	responses = append(responses, "连续自动回合达到安全上限，战斗暂停等待你的下一步。")
	return responses, false, nil
}

func (a *CombatDMAgent) processCurrentActorInput(ctx context.Context, input string, playerControlled bool) (*combatActionEvent, error) {
	if err := a.refreshTurnState(ctx); err != nil {
		return nil, err
	}
	if a.ended {
		return &combatActionEvent{CombatEnded: true, Message: "战斗已经结束。"}, nil
	}
	if a.turnState == nil {
		return &combatActionEvent{Message: "当前回合状态不可用。"}, nil
	}

	if isPureEndTurnInput(input) {
		return a.advanceTurn(ctx, fmt.Sprintf("%s结束了回合。", a.turnState.ActorName))
	}

	intent, err := a.resolveIntent(ctx, input)
	if err != nil {
		a.logger.Warn("Combat intent resolver failed; using fallback",
			zap.String("actor", a.turnState.ActorName),
			zap.Error(err),
		)
		intent = a.fallbackIntent(input)
	}

	if intent.IntentType == combatIntentEndTurn {
		return a.advanceTurn(ctx, fmt.Sprintf("%s结束了回合。", a.turnState.ActorName))
	}
	if intent.IntentType != combatIntentAction {
		msg := a.invalidIntentMessage("无法从输入中匹配可执行动作。")
		if !playerControlled {
			return a.advanceTurn(ctx, fmt.Sprintf("%s犹豫了一下，没有采取有效动作。", a.turnState.ActorName))
		}
		return &combatActionEvent{Intent: intent, Message: msg}, nil
	}

	action, ok := findAvailableAction(a.cachedActions, intent.ActionID)
	if !ok {
		if !playerControlled {
			intent = a.fallbackIntent(input)
			action, ok = findAvailableAction(a.cachedActions, intent.ActionID)
		}
		if !ok {
			msg := a.invalidIntentMessage(fmt.Sprintf("动作 %q 当前不可用。", intent.ActionID))
			if !playerControlled {
				return a.advanceTurn(ctx, fmt.Sprintf("%s没有找到可执行动作，回合结束。", a.turnState.ActorName))
			}
			return &combatActionEvent{Intent: intent, Message: msg}, nil
		}
	}

	if action.RequiresTarget && intent.TargetID == "" && len(intent.TargetIDs) == 0 && len(action.ValidTargetIDs) > 0 {
		intent.TargetID = action.ValidTargetIDs[0]
	}
	if err := validateIntentTarget(intent, action); err != nil {
		if !playerControlled {
			intent.TargetID = firstValidTarget(action)
		}
		if err := validateIntentTarget(intent, action); err != nil {
			msg := a.invalidIntentMessage(err.Error())
			if !playerControlled {
				return a.advanceTurn(ctx, fmt.Sprintf("%s无法选择有效目标，回合结束。", a.turnState.ActorName))
			}
			return &combatActionEvent{Intent: intent, Message: msg}, nil
		}
	}

	req := dndengine.ExecuteTurnActionRequest{
		GameID:    a.gameID,
		ActorID:   a.turnState.ActorID,
		ActionID:  intent.ActionID,
		TargetID:  intent.TargetID,
		TargetIDs: intent.TargetIDs,
	}
	execResult, err := a.dndEngine.ExecuteTurnAction(ctx, req)
	if err != nil {
		if !playerControlled {
			a.logger.Warn("Enemy action execution failed; advancing turn", zap.Error(err))
			return a.advanceTurn(ctx, fmt.Sprintf("%s的行动失败，回合结束。", a.turnState.ActorName))
		}
		return &combatActionEvent{
			ActorID:       a.turnState.ActorID,
			ActorName:     a.turnState.ActorName,
			ActorType:     a.turnState.ActorType,
			OriginalInput: input,
			Intent:        intent,
			Message:       a.invalidIntentMessage(err.Error()),
		}, nil
	}

	event := combatActionEvent{
		ActorID:       a.turnState.ActorID,
		ActorName:     a.turnState.ActorName,
		ActorType:     a.turnState.ActorType,
		OriginalInput: input,
		Intent:        intent,
		ExecuteResult: execResult,
		CombatEnded:   execResult.CombatEnd != nil,
	}

	shouldAdvance := execResult.CombatEnd == nil && (execResult.TurnComplete || containsEndTurnHint(input) || !playerControlled)
	if shouldAdvance {
		next, err := a.dndEngine.NextTurnWithActions(ctx, dndengine.NextTurnRequest{GameID: a.gameID})
		if err != nil {
			return nil, err
		}
		event.NextTurnResult = next
		if next.Turn != nil && next.Turn.CombatEnd != nil {
			event.CombatEnded = true
		}
	}
	if event.CombatEnded {
		a.ended = true
	}

	event.Message = a.narrateEvent(ctx, event)
	a.recordCombatEvent(event)
	if event.CombatEnded {
		a.summary = a.generateCombatSummary(ctx, &event)
		a.endCombatInEngine(ctx)
	}
	a.history = append(a.history, llm.NewAssistantMessage(event.Message, nil))
	a.trimHistoryIfNeeded()
	return &event, nil
}

func (a *CombatDMAgent) advanceTurn(ctx context.Context, fallbackMessage string) (*combatActionEvent, error) {
	actorID, actorName, actorType := model.ID(""), "未知角色", ""
	if a.turnState != nil {
		actorID, actorName, actorType = a.turnState.ActorID, a.turnState.ActorName, a.turnState.ActorType
	}
	next, err := a.dndEngine.NextTurnWithActions(ctx, dndengine.NextTurnRequest{GameID: a.gameID})
	if err != nil {
		return nil, err
	}
	event := combatActionEvent{
		ActorID:        actorID,
		ActorName:      actorName,
		ActorType:      actorType,
		Intent:         ResolvedCombatIntent{IntentType: combatIntentEndTurn, Confidence: 1},
		NextTurnResult: next,
		Message:        fallbackMessage,
	}
	if next.Turn != nil && next.Turn.CombatEnd != nil {
		event.CombatEnded = true
		a.ended = true
	}
	if event.CombatEnded {
		event.Message = a.narrateEvent(ctx, event)
		a.recordCombatEvent(event)
		a.summary = a.generateCombatSummary(ctx, &event)
		a.endCombatInEngine(ctx)
	} else if event.Message != "" {
		a.recordCombatEvent(event)
	}
	a.history = append(a.history, llm.NewAssistantMessage(event.Message, nil))
	return &event, nil
}

func (a *CombatDMAgent) resolveIntent(ctx context.Context, input string) (ResolvedCombatIntent, error) {
	if isPureEndTurnInput(input) {
		return ResolvedCombatIntent{IntentType: combatIntentEndTurn, Confidence: 1}, nil
	}
	if a.llmClient == nil {
		return a.fallbackIntent(input), nil
	}

	payload := map[string]any{
		"actor":       a.turnState,
		"input":       input,
		"battlefield": a.formatBattlefieldSummary(ctx),
		"actions":     actionSummaries(a.cachedActions),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		a.logger.Warn("Failed to marshal combat intent payload", zap.Error(err))
		return a.fallbackIntent(input), nil
	}
	resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			llm.NewSystemMessage(`你是D&D战斗动作解析器。只输出JSON，不要叙事。JSON格式: {"intent_type":"action|end_turn|invalid","action_id":"...","target_id":"...","target_ids":["..."],"confidence":0.0,"reason":"..."}。只能选择输入actions中存在的action_id和valid_target_ids。`),
			llm.NewUserMessage(string(payloadJSON)),
		},
		Temperature: 0,
		MaxTokens:   300,
	})
	if err != nil {
		return ResolvedCombatIntent{}, err
	}

	var intent ResolvedCombatIntent
	if err := json.Unmarshal([]byte(extractJSONObject(resp.Content)), &intent); err != nil {
		return ResolvedCombatIntent{}, err
	}
	if intent.IntentType == "" {
		intent.IntentType = combatIntentInvalid
	}
	return intent, nil
}

func (a *CombatDMAgent) fallbackIntent(input string) ResolvedCombatIntent {
	if isPureEndTurnInput(input) {
		return ResolvedCombatIntent{IntentType: combatIntentEndTurn, Confidence: 1}
	}
	if a.cachedActions == nil {
		return ResolvedCombatIntent{IntentType: combatIntentInvalid, Reason: "no available actions"}
	}

	lower := strings.ToLower(input)
	for _, action := range proactiveActions(a.cachedActions) {
		if strings.Contains(lower, strings.ToLower(action.ID)) || strings.Contains(input, action.Name) {
			return intentFromAction(action, 0.8, "matched action text")
		}
	}
	for _, action := range proactiveActions(a.cachedActions) {
		if action.Category == "attack" || strings.Contains(action.ID, "attack") || strings.Contains(action.Name, "攻击") {
			return intentFromAction(action, 0.5, "fallback attack")
		}
	}
	actions := proactiveActions(a.cachedActions)
	if len(actions) > 0 {
		return intentFromAction(actions[0], 0.3, "fallback first action")
	}
	return ResolvedCombatIntent{IntentType: combatIntentInvalid, Reason: "no proactive actions"}
}

func intentFromAction(action dndengine.AvailableAction, confidence float64, reason string) ResolvedCombatIntent {
	intent := ResolvedCombatIntent{
		IntentType: combatIntentAction,
		ActionID:   action.ID,
		Confidence: confidence,
		Reason:     reason,
	}
	if action.RequiresTarget && len(action.ValidTargetIDs) > 0 {
		intent.TargetID = action.ValidTargetIDs[0]
	}
	return intent
}

func (a *CombatDMAgent) narrateEvent(ctx context.Context, event combatActionEvent) string {
	fallback := fallbackNarrative(event)
	if a.llmClient == nil {
		return fallback
	}

	payload, err := json.Marshal(map[string]any{
		"actor_name":       event.ActorName,
		"actor_type":       event.ActorType,
		"original_input":   event.OriginalInput,
		"intent":           event.Intent,
		"execute_result":   event.ExecuteResult,
		"next_turn_result": event.NextTurnResult,
		"combat_ended":     event.CombatEnded,
		"battlefield":      a.formatBattlefieldSummary(ctx),
	})
	if err != nil {
		a.logger.Warn("Failed to marshal combat narration payload", zap.Error(err))
		return fallback
	}
	resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			llm.NewSystemMessage("你是D&D战斗叙事助手。只能根据输入JSON中的事实写2-4句中文叙事，不得自行计算或编造骰点、伤害、HP。"),
			llm.NewUserMessage(string(payload)),
		},
		Temperature: 0.7,
		MaxTokens:   500,
	})
	if err != nil || strings.TrimSpace(resp.Content) == "" {
		if err != nil {
			a.logger.Warn("Combat narration failed; using fallback", zap.Error(err))
		}
		return fallback
	}
	return strings.TrimSpace(resp.Content)
}

func fallbackNarrative(event combatActionEvent) string {
	var parts []string
	if event.ExecuteResult != nil {
		if event.ExecuteResult.Narrative != "" {
			parts = append(parts, event.ExecuteResult.Narrative)
		} else if event.ExecuteResult.ActionName != "" {
			parts = append(parts, fmt.Sprintf("%s执行了%s。", event.ActorName, event.ExecuteResult.ActionName))
		}
	}
	if event.NextTurnResult != nil && event.NextTurnResult.Turn != nil {
		if event.NextTurnResult.Turn.CombatEnd != nil {
			parts = append(parts, fmt.Sprintf("战斗结束：%s。", event.NextTurnResult.Turn.CombatEnd.Reason))
		} else {
			parts = append(parts, fmt.Sprintf("现在轮到%s行动。", event.NextTurnResult.Turn.ActorName))
		}
	}
	if len(parts) == 0 && event.Message != "" {
		parts = append(parts, event.Message)
	}
	if len(parts) == 0 {
		parts = append(parts, "战斗继续。")
	}
	return strings.Join(parts, "\n")
}

func (a *CombatDMAgent) describeCurrentTurn(ctx context.Context) string {
	if err := a.refreshTurnState(ctx); err != nil {
		a.logger.Warn("Failed to refresh turn state for current turn description", zap.Error(err))
		return "战斗状态暂时不可用。"
	}
	if a.ended {
		return "战斗已经结束。"
	}
	if a.turnState == nil {
		return "战斗进行中，但当前回合信息不可用。"
	}
	return fmt.Sprintf("%s\n当前轮到%s行动。\n%s", a.formatBattlefieldSummary(ctx), a.turnState.ActorName, a.formatAvailableActions())
}

func (a *CombatDMAgent) invalidIntentMessage(reason string) string {
	return fmt.Sprintf("%s\n当前可用动作：\n%s", reason, a.formatAvailableActions())
}

// ============================================================================
// 内层 ReAct 循环
// ============================================================================

// executeReActLoop 内层 LLM tool-calling 循环
// 循环：调用 LLM → 如果有 ToolCalls 则执行工具 → 继续
// 直到 LLM yield（返回纯文本无 ToolCalls）
func (a *CombatDMAgent) executeReActLoop(ctx context.Context) (string, error) {
	for i := 0; i < maxReActIterations; i++ {
		a.logger.Debug("ReAct iteration",
			zap.Int("iteration", i),
			zap.Int("historyLen", len(a.history)),
		)

		resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
			Messages:    a.history,
			Tools:       a.toolSchemas,
			Temperature: 0.7,
		})
		if err != nil {
			return "", fmt.Errorf("LLM call failed at iteration %d: %w", i, err)
		}

		// LLM 请求执行工具
		if len(resp.ToolCalls) > 0 {
			// 记录 assistant 消息（含 ToolCalls）
			a.history = append(a.history, llm.NewAssistantMessage(resp.Content, resp.ToolCalls))

			// 执行所有工具调用
			results := a.registry.ExecuteTools(ctx, resp.ToolCalls)

			// 记录工具结果到历史
			for _, r := range results {
				a.history = append(a.history, llm.NewToolMessage(r.Content, r.ToolCallID))
			}

			// 从 dnd-core 引擎直接查询回合状态
			if err := a.refreshTurnState(ctx); err != nil {
				return "", err
			}

			// 历史过长时截断
			a.trimHistoryIfNeeded()

			// 执行工具后打印日志
			a.logHistoryAfterComplete(i, resp)

			continue
		}

		// LLM yield（纯文本，无 ToolCalls）
		a.history = append(a.history, llm.NewAssistantMessage(resp.Content, nil))

		// yield 后打印日志（此时 history 已包含 LLM 回复）
		a.logHistoryAfterComplete(i, resp)

		return resp.Content, nil
	}

	// 达到最大迭代次数
	fallbackMsg := "（战斗处理达到最大步数限制，请继续描述你的行动）"
	a.history = append(a.history, llm.NewAssistantMessage(fallbackMsg, nil))
	return fallbackMsg, nil
}

// ============================================================================
// 回合状态管理（直接查询 dnd-core 引擎）
// ============================================================================

// refreshTurnState 从 dnd-core 引擎直接查询当前回合状态
// 在每次工具执行后调用，替代从工具结果 JSON 中解析
func (a *CombatDMAgent) refreshTurnState(ctx context.Context) error {
	// 1. 查询当前战斗状态
	combatResult, err := a.dndEngine.GetCurrentCombat(ctx, dndengine.GetCurrentCombatRequest{
		GameID: a.gameID,
	})
	if err != nil {
		if errors.Is(err, dndengine.ErrCombatNotActive) {
			// 战斗不活跃（可能已结束）
			a.ended = true
			a.logger.Info("Combat ended (GetCurrentCombat returned no active combat)", zap.Error(err))
			return nil
		}
		return err
	}
	if combatResult.Combat.Status != model.CombatStatusActive {
		a.ended = true
		a.logger.Info("Combat ended (status not active)",
			zap.String("status", string(combatResult.Combat.Status)),
		)
		return nil
	}

	// 2. 获取当前行动者 ID
	currentTurn := combatResult.Combat.CurrentTurn
	if currentTurn == nil {
		a.logger.Warn("Combat active but CurrentTurn is nil")
		return fmt.Errorf("combat active but current turn is nil")
	}

	// 3. 查询当前角色的可用动作（同时获取 ActorType）
	actions, err := a.dndEngine.GetAvailableActions(ctx, dndengine.GetAvailableActionsRequest{
		GameID:  a.gameID,
		ActorID: currentTurn.ActorID,
	})
	if err != nil {
		// fallback: 仅用 combat info 中的信息
		a.logger.Warn("Failed to get available actions, using basic turn info", zap.Error(err))
		a.turnState = &turnStateCache{
			ActorID:   currentTurn.ActorID,
			ActorName: currentTurn.ActorName,
		}
		a.cachedActions = nil
		return err
	}

	a.turnState = &turnStateCache{
		ActorID:   currentTurn.ActorID,
		ActorName: actions.ActorName,
		ActorType: actions.ActorType,
	}
	a.cachedActions = actions

	a.logger.Debug("Turn state refreshed from engine",
		zap.String("actor", actions.ActorName),
		zap.String("type", actions.ActorType),
	)
	return nil
}

// isPlayerTurn 判断当前是否为玩家回合
func (a *CombatDMAgent) isPlayerTurn() bool {
	if a.turnState == nil {
		// 无法判断时，保守策略：yield 给玩家
		return true
	}
	return a.turnState.ActorType == string(model.ActorTypePC)
}

// ============================================================================
// EnemyAI 意图生成
// ============================================================================

// generateEnemyIntent 调用 EnemyAIAgent 为当前敌人生成行动意图
func (a *CombatDMAgent) generateEnemyIntent(ctx context.Context) (string, error) {
	if a.turnState == nil {
		return "攻击最近的敌人", nil
	}

	battlefield := a.formatBattlefieldSummary(ctx)
	actionsText := a.formatAvailableActions()

	if a.enemyAI == nil {
		return "采取最直接的攻击行动", nil
	}

	return a.enemyAI.GenerateIntent(
		ctx,
		a.turnState.ActorName,
		a.turnState.ActorType,
		battlefield,
		actionsText,
	)
}

// formatBattlefieldSummary 从 dnd-core 引擎查询并格式化战场态势
func (a *CombatDMAgent) formatBattlefieldSummary(ctx context.Context) string {
	summary, err := a.dndEngine.GetStateSummary(ctx, a.gameID)
	if err != nil || summary.ActiveCombat == nil {
		return "战斗进行中"
	}

	combat := summary.ActiveCombat
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("第%d轮，当前行动者: %s\n", combat.Round, combat.CurrentActor))
	for _, c := range combat.Combatants {
		status := ""
		if c.IsDefeated {
			status = " [已倒下]"
		}
		if len(c.Conditions) > 0 {
			status += " " + strings.Join(c.Conditions, ",")
		}
		sb.WriteString(fmt.Sprintf("- %s (%s): HP %d/%d, AC %d%s\n",
			c.Name, c.Type, c.HP, c.MaxHP, c.AC, status))
	}
	return sb.String()
}

// formatAvailableActions 从缓存的可用动作中格式化文本
func (a *CombatDMAgent) formatAvailableActions() string {
	if a.cachedActions == nil {
		return "（可用动作信息不可用）"
	}

	var sb strings.Builder
	formatActions := func(label string, actions []dndengine.AvailableAction) {
		if len(actions) == 0 {
			return
		}
		sb.WriteString(label + ":\n")
		for _, act := range actions {
			desc := act.Name
			if act.Range != "" {
				desc += " (" + act.Range + ")"
			}
			if act.DamagePreview != "" {
				desc += " " + act.DamagePreview
			}
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", act.ID, desc))
		}
	}

	if a.cachedActions.Movement.Available {
		sb.WriteString(fmt.Sprintf("移动: 剩余 %d 尺\n", a.cachedActions.Movement.RemainingFeet))
	}
	formatActions("动作", a.cachedActions.Actions)
	formatActions("附赠动作", a.cachedActions.BonusActions)
	formatActions("自由动作", a.cachedActions.FreeActions)
	return sb.String()
}

func (a *CombatDMAgent) recordCombatEvent(event combatActionEvent) {
	if event.Message == "" && event.ExecuteResult == nil && event.NextTurnResult == nil {
		return
	}
	a.events = append(a.events, event)
}

func (a *CombatDMAgent) generateCombatSummary(ctx context.Context, finalEvent *combatActionEvent) string {
	fallback := a.fallbackCombatSummary(finalEvent)
	if a.llmClient == nil {
		return fallback
	}

	eventLines := make([]string, 0, len(a.events))
	for _, event := range a.events {
		eventLines = append(eventLines, summarizeCombatEvent(event))
	}
	payload, err := json.Marshal(map[string]any{
		"battlefield":  a.summaryBattlefield(ctx),
		"events":       eventLines,
		"final_event":  summarizeCombatEventPtr(finalEvent),
		"combat_ended": a.ended,
	})
	if err != nil {
		a.logger.Warn("Failed to marshal combat summary payload", zap.Error(err))
		return fallback
	}

	resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			llm.NewSystemMessage("你是D&D战斗记录员。只根据输入JSON中的事实，用简短中文条目生成战后摘要，必须以“[战斗摘要]”开头，不得编造奖励、掉落、经验或任务进展。"),
			llm.NewUserMessage(string(payload)),
		},
		Temperature: 0.2,
		MaxTokens:   500,
	})
	if err != nil || strings.TrimSpace(resp.Content) == "" {
		if err != nil {
			a.logger.Warn("Combat summary generation failed; using fallback", zap.Error(err))
		}
		return fallback
	}

	summary := strings.TrimSpace(resp.Content)
	if !strings.HasPrefix(summary, "[战斗摘要]") {
		summary = "[战斗摘要]\n" + summary
	}
	return summary
}

func (a *CombatDMAgent) fallbackCombatSummary(finalEvent *combatActionEvent) string {
	var sb strings.Builder
	sb.WriteString("[战斗摘要]\n")
	if finalEvent != nil {
		if finalEvent.ExecuteResult != nil && finalEvent.ExecuteResult.CombatEnd != nil {
			sb.WriteString(fmt.Sprintf("- 战斗结束：%s，胜利方：%s。\n", finalEvent.ExecuteResult.CombatEnd.Reason, finalEvent.ExecuteResult.CombatEnd.Winners))
		} else if finalEvent.NextTurnResult != nil && finalEvent.NextTurnResult.Turn != nil && finalEvent.NextTurnResult.Turn.CombatEnd != nil {
			end := finalEvent.NextTurnResult.Turn.CombatEnd
			sb.WriteString(fmt.Sprintf("- 战斗结束：%s，胜利方：%s。\n", end.Reason, end.Winners))
		}
	}
	if len(a.events) == 0 {
		sb.WriteString("- 战斗已结束，具体过程记录不足。\n")
	} else {
		sb.WriteString(fmt.Sprintf("- 战斗共记录 %d 个关键事件。\n", len(a.events)))
		limit := len(a.events)
		if limit > 6 {
			limit = 6
			sb.WriteString("- 以下为最后 6 个关键事件：\n")
		}
		start := len(a.events) - limit
		for _, event := range a.events[start:] {
			line := summarizeCombatEvent(event)
			if line != "" {
				sb.WriteString("- " + line + "\n")
			}
		}
	}
	sb.WriteString("- 后续叙事可从战场检查、伤势处理、战利品确认或继续探索展开。")
	return strings.TrimSpace(sb.String())
}

func (a *CombatDMAgent) summaryBattlefield(ctx context.Context) string {
	summary, err := a.dndEngine.GetStateSummary(ctx, a.gameID)
	if err == nil && summary.ActiveCombat != nil {
		return a.formatBattlefieldSummary(ctx)
	}

	eventLines := make([]string, 0, len(a.events))
	for _, event := range a.events {
		line := summarizeCombatEvent(event)
		if line != "" {
			eventLines = append(eventLines, line)
		}
	}
	if len(eventLines) == 0 {
		return "战斗已结束；没有可用的活跃战场快照。"
	}
	return "战斗已结束；活跃战场快照已关闭，以下为缓存事件：\n" + strings.Join(eventLines, "\n")
}

func (a *CombatDMAgent) endCombatInEngine(ctx context.Context) {
	if err := a.dndEngine.EndCombat(ctx, dndengine.EndCombatRequest{GameID: a.gameID}); err != nil {
		a.logger.Warn("Failed to end combat in dnd-core; phase may already be exploration",
			zap.String("gameID", string(a.gameID)),
			zap.Error(err),
		)
	}
}

func summarizeCombatEventPtr(event *combatActionEvent) string {
	if event == nil {
		return ""
	}
	return summarizeCombatEvent(*event)
}

func summarizeCombatEvent(event combatActionEvent) string {
	actorName := event.ActorName
	if actorName == "" {
		actorName = "未知角色"
	}
	var parts []string
	if event.ExecuteResult != nil {
		if event.ExecuteResult.ActionName != "" {
			parts = append(parts, fmt.Sprintf("%s执行%s", actorName, event.ExecuteResult.ActionName))
		}
		if event.ExecuteResult.Narrative != "" {
			parts = append(parts, event.ExecuteResult.Narrative)
		}
		if event.ExecuteResult.CombatEnd != nil {
			parts = append(parts, fmt.Sprintf("战斗结束:%s/%s", event.ExecuteResult.CombatEnd.Reason, event.ExecuteResult.CombatEnd.Winners))
		}
	}
	if event.NextTurnResult != nil && event.NextTurnResult.Turn != nil {
		if event.NextTurnResult.Turn.CombatEnd != nil {
			end := event.NextTurnResult.Turn.CombatEnd
			parts = append(parts, fmt.Sprintf("战斗结束:%s/%s", end.Reason, end.Winners))
		} else if event.NextTurnResult.Turn.ActorName != "" {
			parts = append(parts, fmt.Sprintf("下一回合:%s", event.NextTurnResult.Turn.ActorName))
		}
	}
	if len(parts) == 0 && event.Message != "" {
		parts = append(parts, event.Message)
	}
	return strings.TrimSpace(strings.Join(parts, "；"))
}

// ============================================================================
// 系统提示词与工具 Schema 构建
// ============================================================================

// buildSystemPrompt 构建系统提示词
func (a *CombatDMAgent) buildSystemPrompt() string {
	// 构建工具描述列表
	tools := a.registry.GetByAgent(CombatDMAgentName)
	type toolDesc struct {
		Name        string
		Description string
	}
	toolDescs := make([]toolDesc, len(tools))
	for i, t := range tools {
		toolDescs[i] = toolDesc{Name: t.Name(), Description: t.Description()}
	}

	rendered, err := prompt.LoadAndRender("combat_dm_system.md", map[string]any{
		"GameID":         string(a.gameID),
		"PlayerID":       string(a.playerID),
		"AvailableTools": toolDescs,
	})
	if err != nil {
		a.logger.Error("Failed to load combat DM system prompt", zap.Error(err))
		return fmt.Sprintf("你是D&D 5e战斗DM。游戏ID: %s, 玩家ID: %s。通过工具调用管理战斗。", a.gameID, a.playerID)
	}
	return rendered
}

// buildToolSchemas 构建 LLM function calling 格式的工具 schema
func (a *CombatDMAgent) buildToolSchemas() []map[string]any {
	tools := a.registry.GetByAgent(CombatDMAgentName)
	schemas := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		schemas = append(schemas, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  t.ParametersSchema(),
			},
		})
	}
	return schemas
}

// ============================================================================
// 历史管理
// ============================================================================

// trimHistoryIfNeeded 当历史过长时截断
// 保留系统提示词 + 最近 N 条消息
func (a *CombatDMAgent) trimHistoryIfNeeded() {
	if len(a.history) <= maxHistoryMessages {
		return
	}

	a.logger.Info("Trimming combat history",
		zap.Int("before", len(a.history)),
		zap.Int("keepRecent", keepRecentMessages),
	)

	// 保留第一条（系统提示词）+ 最后 N 条
	newHistory := make([]llm.Message, 0, 1+keepRecentMessages)
	newHistory = append(newHistory, a.history[0]) // system prompt
	newHistory = append(newHistory, a.history[len(a.history)-keepRecentMessages:]...)
	a.history = newHistory
}

// logHistoryAfterComplete 在 LLM 调用后打印完整 history（含 LLM 返回结果）
func (a *CombatDMAgent) logHistoryAfterComplete(iteration int, resp *llm.CompletionResponse) {
	toolInfo := ""
	if len(resp.ToolCalls) > 0 {
		var names []string
		for _, tc := range resp.ToolCalls {
			names = append(names, tc.Name)
		}
		toolInfo = fmt.Sprintf(" [tool_calls: %s]", strings.Join(names, ", "))
	}

	a.logger.Info("[CombatDM] LLM response",
		zap.Int("iteration", iteration),
		zap.String("content", truncateStr(resp.Content, 300)),
		zap.String("meta", toolInfo),
	)

	for i, msg := range a.history {
		content := msg.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		tcInfo := ""
		if len(msg.ToolCalls) > 0 {
			var names []string
			for _, tc := range msg.ToolCalls {
				names = append(names, tc.Name)
			}
			tcInfo = fmt.Sprintf(" [tool_calls: %s]", strings.Join(names, ", "))
		}
		a.logger.Info("[CombatDM] history",
			zap.Int("iteration", iteration),
			zap.Int("index", i),
			zap.String("role", string(msg.Role)),
			zap.String("meta", tcInfo),
			zap.String("content", content),
		)
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func hasProactiveActions(actions *dndengine.AvailableActionsResult) bool {
	return len(proactiveActions(actions)) > 0
}

func proactiveActions(actions *dndengine.AvailableActionsResult) []dndengine.AvailableAction {
	if actions == nil {
		return nil
	}
	all := make([]dndengine.AvailableAction, 0, len(actions.Actions)+len(actions.BonusActions)+len(actions.FreeActions))
	all = append(all, actions.Actions...)
	all = append(all, actions.BonusActions...)
	all = append(all, actions.FreeActions...)
	return all
}

func findAvailableAction(actions *dndengine.AvailableActionsResult, actionID string) (dndengine.AvailableAction, bool) {
	for _, action := range proactiveActions(actions) {
		if action.ID == actionID {
			return action, true
		}
	}
	return dndengine.AvailableAction{}, false
}

func validateIntentTarget(intent ResolvedCombatIntent, action dndengine.AvailableAction) error {
	if !action.RequiresTarget {
		return nil
	}
	if intent.TargetID == "" && len(intent.TargetIDs) == 0 {
		return fmt.Errorf("动作 %q 需要目标。", action.ID)
	}
	if intent.TargetID != "" && !idInList(intent.TargetID, action.ValidTargetIDs) {
		return fmt.Errorf("目标 %q 不是动作 %q 的合法目标。", intent.TargetID, action.ID)
	}
	for _, id := range intent.TargetIDs {
		if !idInList(id, action.ValidTargetIDs) {
			return fmt.Errorf("目标 %q 不是动作 %q 的合法目标。", id, action.ID)
		}
	}
	return nil
}

func firstValidTarget(action dndengine.AvailableAction) model.ID {
	if len(action.ValidTargetIDs) == 0 {
		return ""
	}
	return action.ValidTargetIDs[0]
}

func idInList(id model.ID, ids []model.ID) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func isPureEndTurnInput(input string) bool {
	normalized := strings.TrimSpace(strings.ToLower(input))
	if normalized == "" {
		return false
	}
	endPhrases := []string{"结束回合", "跳过", "待机", "pass", "end turn", "结束"}
	actionHints := []string{"攻击", "施放", "释放", "使用", "冲刺", "撤离", "闪避", "协助", "躲藏", "搜索", "attack", "cast", "use"}
	hasEnd := false
	for _, phrase := range endPhrases {
		if strings.Contains(normalized, phrase) {
			hasEnd = true
			break
		}
	}
	if !hasEnd {
		return false
	}
	for _, hint := range actionHints {
		if strings.Contains(normalized, hint) {
			return false
		}
	}
	return true
}

func containsEndTurnHint(input string) bool {
	normalized := strings.TrimSpace(strings.ToLower(input))
	for _, phrase := range []string{"结束回合", "跳过", "pass", "end turn"} {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func extractJSONObject(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		return content
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1]
	}
	return content
}

func actionSummaries(actions *dndengine.AvailableActionsResult) []map[string]any {
	summaries := make([]map[string]any, 0)
	for _, action := range proactiveActions(actions) {
		validTargets := make([]string, 0, len(action.ValidTargetIDs))
		for _, id := range action.ValidTargetIDs {
			validTargets = append(validTargets, string(id))
		}
		summaries = append(summaries, map[string]any{
			"id":               action.ID,
			"name":             action.Name,
			"category":         action.Category,
			"cost_type":        action.CostType,
			"requires_target":  action.RequiresTarget,
			"target_type":      action.TargetType,
			"valid_target_ids": validTargets,
			"range":            action.Range,
			"damage_preview":   action.DamagePreview,
			"resource_cost":    action.ResourceCost,
			"description":      action.Description,
		})
	}
	return summaries
}
