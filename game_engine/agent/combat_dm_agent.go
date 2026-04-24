package agent

import (
	"context"
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

// Initialize 初始化战斗会话
// 添加系统提示词到历史，然后运行第一轮循环（无玩家输入）
func (a *CombatDMAgent) Initialize(ctx context.Context) (*CombatAgentResult, error) {
	a.logger.Info("CombatDMAgent initializing",
		zap.String("gameID", string(a.gameID)),
	)

	// 构建系统提示词
	systemPrompt := a.buildSystemPrompt()
	a.history = append(a.history, llm.NewSystemMessage(systemPrompt))

	// 运行第一轮（无用户输入，LLM 应主动调用 next_turn_with_actions 获取初始状态）
	return a.RunLoop(ctx, "")
}

// RunLoop 主运行循环（两层循环架构）
//
// 外层循环：Go 框架层回合路由
//  1. 将 input 加入 history（如果非空）
//  2. 运行内层 ReAct 循环 → LLM yield 出文本
//  3. 检查：战斗结束？→ return
//  4. 检查：当前是玩家回合？→ return（yield 给玩家）
//  5. 当前是敌人回合 → 调用 EnemyAI → 将意图作为 UserMessage 加入 history → 回到步骤 2
func (a *CombatDMAgent) RunLoop(ctx context.Context, input string) (*CombatAgentResult, error) {
	if input != "" {
		a.history = append(a.history, llm.NewUserMessage(input))
	}

	var allResponses []string
	enemyChainCount := 0

	for {
		// 内层 ReAct 循环
		response, err := a.executeReActLoop(ctx)
		if err != nil {
			return nil, fmt.Errorf("ReAct loop error: %w", err)
		}

		allResponses = append(allResponses, response)

		// 强制刷新战斗状态，防御大模型跳过工具调用直接返回结束文本的情况
		a.refreshTurnState(ctx)

		// 战斗结束
		if a.ended {
			return &CombatAgentResult{
				Response:    strings.Join(allResponses, "\n\n"),
				CombatEnded: true,
			}, nil
		}

		// 玩家回合 → yield
		if a.isPlayerTurn() {
			return &CombatAgentResult{
				Response:    strings.Join(allResponses, "\n\n"),
				CombatEnded: false,
			}, nil
		}

		// 敌人/NPC 回合 → 生成意图，喂回 LLM
		enemyChainCount++
		if enemyChainCount >= maxEnemyTurnChain {
			a.logger.Warn("Max enemy turn chain reached, forcing yield",
				zap.Int("chainCount", enemyChainCount),
			)
			return &CombatAgentResult{
				Response:    strings.Join(allResponses, "\n\n"),
				CombatEnded: false,
			}, nil
		}

		intent, err := a.generateEnemyIntent(ctx)
		if err != nil {
			a.logger.Error("Failed to generate enemy intent", zap.Error(err))
			intent = "发呆，不采取任何行动"
		}

		// 将敌人意图作为 UserMessage 注入历史
		actorName := "未知角色"
		if a.turnState != nil {
			actorName = a.turnState.ActorName
		}
		intentMsg := fmt.Sprintf("[%s的行动] %s", actorName, intent)
		a.history = append(a.history, llm.NewUserMessage(intentMsg))

		a.logger.Info("Enemy intent injected",
			zap.String("actor", actorName),
			zap.String("intent", intent),
		)
	}
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
			a.refreshTurnState(ctx)

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
func (a *CombatDMAgent) refreshTurnState(ctx context.Context) {
	// 1. 查询当前战斗状态
	combatResult, err := a.dndEngine.GetCurrentCombat(ctx, dndengine.GetCurrentCombatRequest{
		GameID: a.gameID,
	})
	if err != nil {
		// 战斗不活跃（可能已结束）
		a.ended = true
		a.logger.Info("Combat ended (GetCurrentCombat returned error)", zap.Error(err))
		return
	}
	if combatResult.Combat.Status != model.CombatStatusActive {
		a.ended = true
		a.logger.Info("Combat ended (status not active)",
			zap.String("status", string(combatResult.Combat.Status)),
		)
		return
	}

	// 2. 获取当前行动者 ID
	currentTurn := combatResult.Combat.CurrentTurn
	if currentTurn == nil {
		a.logger.Warn("Combat active but CurrentTurn is nil")
		return
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
		return
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
