package gameengine

import (
	"context"

	"go.uber.org/zap"

	"github.com/zwh8800/cdndv2/game_engine/agent"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/tool"
	dndengine "github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ============================================================================
// CombatSession — 独立战斗会话（薄包装器）
// ============================================================================

// CombatSession 管理一场完整战斗的独立会话
// 在战斗阶段接管 ProcessInput，将所有逻辑委托给 CombatDMAgent
type CombatSession struct {
	gameID   model.ID
	playerID model.ID
	combatDM *agent.CombatDMAgent
	logger   *zap.Logger
}

// CombatResult 战斗处理结果
type CombatResult struct {
	Response    string // 返回给玩家的叙述文本
	CombatEnded bool   // 战斗是否结束
	Summary     string // 战斗结束时的摘要（如有）
}

// NewCombatSession 创建新的战斗会话
func NewCombatSession(
	gameID, playerID model.ID,
	llmClient llm.LLMClient,
	registry *tool.ToolRegistry,
	dndEngine *dndengine.Engine,
	enemyAI *agent.EnemyAIAgent,
	logger *zap.Logger,
) *CombatSession {
	return &CombatSession{
		gameID:   gameID,
		playerID: playerID,
		combatDM: agent.NewCombatDMAgent(gameID, playerID, llmClient, registry, dndEngine, enemyAI, logger),
		logger:   logger,
	}
}

// IsWaitingForPlayer 返回当前是否在等待玩家输入
func (cs *CombatSession) IsWaitingForPlayer() bool {
	return !cs.combatDM.IsEnded()
}

// Initialize 初始化战斗会话
// 添加系统提示词，运行第一轮循环获取初始战斗状态
func (cs *CombatSession) Initialize(ctx context.Context) (*CombatResult, error) {
	cs.logger.Info("CombatSession initializing",
		zap.String("gameID", string(cs.gameID)),
	)

	result, err := cs.combatDM.Initialize(ctx)
	if err != nil {
		return nil, err
	}

	return &CombatResult{
		Response:    result.Response,
		CombatEnded: result.CombatEnded,
	}, nil
}

// ProcessInput 处理战斗中的玩家输入
func (cs *CombatSession) ProcessInput(ctx context.Context, input string) (*CombatResult, error) {
	cs.logger.Info("CombatSession processing input",
		zap.String("input", input),
	)

	result, err := cs.combatDM.RunLoop(ctx, input)
	if err != nil {
		return nil, err
	}

	return &CombatResult{
		Response:    result.Response,
		CombatEnded: result.CombatEnded,
	}, nil
}
