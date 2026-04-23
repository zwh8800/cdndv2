package gameengine

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/zwh8800/cdndv2/game_engine/agent"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/llm/openai"
	"github.com/zwh8800/cdndv2/game_engine/tool"
)

// EngineConfig 引擎配置
type EngineConfig struct {
	// DND引擎配置
	DNDEngineConfig engine.Config

	// LLM配置
	LLMConfig openai.OpenAIConfig

	// 最大迭代次数
	MaxIterations int

	// 单次 ProcessInput 请求超时时间（默认 3 分钟）
	RequestTimeout time.Duration

	// OpenAI API密钥（如果LLMConfig中没有设置）
	OpenAIAPIKey string

	// 日志级别 (debug, info, warn, error)，默认为 info
	// 可以通过环境变量 LOG_LEVEL 覆盖
	LogLevel string
}

// GameEngine 游戏引擎
type GameEngine struct {
	dndEngine *engine.Engine
	reactLoop *ReActLoop
	registry  *tool.ToolRegistry
	mainAgent *agent.MainAgent
	llmClient llm.LLMClient
	enemyAI   *agent.EnemyAIAgent // 敌人AI Agent，供战斗session共享
	config    EngineConfig
	logger    *zap.Logger
}

// NewGameEngine 创建新的游戏引擎
func NewGameEngine(cfg EngineConfig) (*GameEngine, error) {
	// 初始化日志器
	logLevel := cfg.LogLevel
	if logLevel == "" {
		logLevel = "info" // 默认 info 级别
	}
	logger, err := initLogger(logLevel)
	if err != nil {
		// 如果初始化失败，使用 Nop logger
		logger = zap.NewNop()
	}

	logger.Info("Initializing GameEngine",
		zap.String("logLevel", logLevel),
		zap.Int("maxIterations", cfg.MaxIterations),
	)

	// 创建D&D引擎
	dndEngine, err := engine.New(cfg.DNDEngineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dnd engine: %w", err)
	}

	// 创建Tool注册中心
	registry := tool.NewToolRegistry()
	registry.SetLogger(logger)

	// 创建LLM客户端
	var llmClient llm.LLMClient

	if cfg.LLMConfig.APIKey == "" && cfg.OpenAIAPIKey != "" {
		cfg.LLMConfig.APIKey = cfg.OpenAIAPIKey
	}

	llmClient, err = openai.NewOpenAIClient(cfg.LLMConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// 设置日志器
	if openAIClient, ok := llmClient.(*openai.OpenAIClient); ok {
		openAIClient.SetLogger(logger)
	}

	// 注册Agent及其工具
	registerAgentTools(registry, dndEngine)
	subAgents := createSubAgents(registry, llmClient)

	// 为子Agent设置日志器
	for _, sa := range subAgents {
		if sl, ok := sa.(interface{ SetLogger(*zap.Logger) }); ok {
			sl.SetLogger(logger)
		}
	}

	// 创建路由Agent（默认禁用，由 MainAgent 通过 delegate_task 直接委派）
	// delegate_task 已覆盖全部 11 个 SubAgent，RouterAgent 作为额外 LLM 调用已不再必要
	// 如需启用，可传入 router 实例
	var router *agent.RouterAgent
	// router = createRouter(llmClient, subAgents)
	// if router != nil {
	// 	router.SetLogger(logger)
	// }

	// 创建主Agent
	mainAgent := agent.NewMainAgent(registry, llmClient)
	mainAgent.SetLogger(logger)

	// 创建敌人AI Agent（供战斗session共享）
	enemyAI := agent.NewEnemyAIAgent(llmClient, logger)

	// 创建ReAct循环
	maxIter := cfg.MaxIterations
	if maxIter == 0 {
		maxIter = 10
	}

	requestTimeout := cfg.RequestTimeout
	if requestTimeout == 0 {
		requestTimeout = 3 * time.Minute
	}
	cfg.RequestTimeout = requestTimeout // 确保 EngineConfig 中也保存了实际使用的值

	reactLoop := NewReActLoop(
		dndEngine,
		mainAgent,
		router, // nil = 跳过 Route 阶段，直接进入 Think
		subAgents,
		registry,
		llmClient,
		maxIter,
	)
	reactLoop.SetLogger(logger)

	// 配置上下文压缩器
	compressor := llm.DefaultContextCompressor(llmClient)
	if cfg.LLMConfig.ContextWindowSize > 0 {
		compressor.ContextWindowSize = cfg.LLMConfig.ContextWindowSize
	}
	compressor.SetToolReadOnlyChecker(registry)
	reactLoop.SetCompressor(compressor)

	return &GameEngine{
		dndEngine: dndEngine,
		reactLoop: reactLoop,
		registry:  registry,
		mainAgent: mainAgent,
		llmClient: llmClient,
		enemyAI:   enemyAI,
		config:    cfg,
		logger:    logger,
	}, nil
}

// initLogger 初始化 zap 日志器
func initLogger(level string) (*zap.Logger, error) {
	lvl := zapcore.InfoLevel
	if level != "" {
		if err := lvl.UnmarshalText([]byte(level)); err != nil {
			lvl = zapcore.InfoLevel
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		lvl,
	)

	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)), nil
}

// GetDNDEngine 获取D&D引擎实例
func (ge *GameEngine) GetDNDEngine() *engine.Engine {
	return ge.dndEngine
}

// GetRegistry 获取Tool注册中心
func (ge *GameEngine) GetRegistry() *tool.ToolRegistry {
	return ge.registry
}

// GetMainAgent 获取主Agent
func (ge *GameEngine) GetMainAgent() *agent.MainAgent {
	return ge.mainAgent
}

// GetLLMClient 获取LLM客户端
func (ge *GameEngine) GetLLMClient() llm.LLMClient {
	return ge.llmClient
}

// SetLLMClient 设置LLM客户端
func (ge *GameEngine) SetLLMClient(client llm.LLMClient) {
	ge.llmClient = client
}

// SetMainAgent 设置主Agent
func (ge *GameEngine) SetMainAgent(mainAgent *agent.MainAgent) {
	ge.mainAgent = mainAgent
	if ge.reactLoop != nil {
		ge.reactLoop.mainAgent = mainAgent
	}
}

// NewGame 创建新游戏
func (ge *GameEngine) NewGame(ctx context.Context, name, description string) (*GameSession, error) {
	result, err := ge.dndEngine.NewGame(ctx, engine.NewGameRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create game: %w", err)
	}

	return &GameSession{
		ID:        result.Game.ID,
		Engine:    ge,
		reactLoop: ge.reactLoop,
	}, nil
}

// LoadGame 加载游戏
func (ge *GameEngine) LoadGame(ctx context.Context, gameID model.ID) (*GameSession, error) {
	_, err := ge.dndEngine.LoadGame(ctx, engine.LoadGameRequest{GameID: gameID})
	if err != nil {
		return nil, fmt.Errorf("failed to load game: %w", err)
	}

	return &GameSession{
		ID:     gameID,
		Engine: ge,
	}, nil
}

// ProcessInput 处理玩家输入
func (ge *GameEngine) ProcessInput(ctx context.Context, session *GameSession, input string) (string, error) {
	// === 战斗拦截路由 ===

	// 1. 如果有活跃的战斗session，将输入交给它处理
	if session.combatSession != nil {
		result, err := session.combatSession.ProcessInput(ctx, input)
		if err != nil {
			return "", err
		}

		if result.CombatEnded {
			// 将战斗摘要注入主历史
			if result.Summary != "" {
				session.reactLoop.state.History = append(
					session.reactLoop.state.History,
					llm.NewAssistantMessage(result.Summary, nil),
				)
			}
			session.combatSession = nil // 释放战斗session
			ge.logger.Info("Combat ended, control returned to MainAgent")
		}
		return result.Response, nil
	}

	// 2. 检查游戏phase是否进入了战斗（由之前的MainAgent触发）
	phase, _ := ge.dndEngine.GetPhase(ctx, session.ID)
	if phase == model.PhaseCombat {
		ge.logger.Info("Detected combat phase, creating CombatSession")
		session.combatSession = NewCombatSession(
			session.ID,
			session.PlayerID,
			ge.llmClient,
			ge.registry,
			ge.dndEngine,
			ge.enemyAI,
			ge.logger,
		)
		result, err := session.combatSession.Initialize(ctx)
		if err != nil {
			ge.logger.Error("Failed to initialize CombatSession", zap.Error(err))
			session.combatSession = nil // 初始化失败，回退到正常流程
		} else {
			if result.CombatEnded {
				if result.Summary != "" {
					session.reactLoop.state.History = append(
						session.reactLoop.state.History,
						llm.NewAssistantMessage(result.Summary, nil),
					)
				}
				session.combatSession = nil
				return result.Response, nil
			}

			// 初始化成功且战斗进行中。如果正在等待玩家输入，
			// 将原始输入作为玩家的第一个战斗指令尝试转发
			if session.combatSession.IsWaitingForPlayer() && input != "" {
				actionResult, err := session.combatSession.ProcessInput(ctx, input)
				if err != nil {
					// 转发失败，只返回初始化描述，玩家可以重新输入
					ge.logger.Warn("Failed to forward input to combat session", zap.Error(err))
					return result.Response, nil
				}
				// 合并初始化描述与动作结果
				response := result.Response + "\n\n---\n\n" + actionResult.Response
				if actionResult.CombatEnded {
					if actionResult.Summary != "" {
						session.reactLoop.state.History = append(
							session.reactLoop.state.History,
							llm.NewAssistantMessage(actionResult.Summary, nil),
						)
					}
					session.combatSession = nil
				}
				return response, nil
			}

			return result.Response, nil
		}
	}

	// === 正常路径：MainAgent ReActLoop ===

	// 在新一轮输入开始时，应用上一轮触发的后台压缩结果
	if session.reactLoop.compressor != nil {
		if compressed := session.reactLoop.compressor.ApplyCompressedIfReady(); compressed != nil {
			ge.logger.Info("Applied async compressed history",
				zap.Int("beforeLen", len(session.reactLoop.state.History)),
				zap.Int("afterLen", len(compressed)),
			)
			session.reactLoop.state.History = compressed
		}
	}

	// 保存历史长度，用于出错时回滚
	historyLenBefore := len(session.reactLoop.state.History)

	// 添加玩家输入到历史
	session.reactLoop.state.History = append(
		session.reactLoop.state.History,
		llm.NewUserMessage(input),
	)

	// 为本次请求设置超时，防止单次输入无限挂起
	timeout := ge.config.RequestTimeout
	if timeout == 0 {
		timeout = 3 * time.Minute
	}
	processCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ge.logger.Info("ProcessInput starting",
		zap.String("input", input),
		zap.Duration("timeout", timeout),
		zap.Int("historyLenBefore", historyLenBefore),
	)

	// 执行ReAct循环
	err := session.reactLoop.Run(processCtx, input, session.ID, session.PlayerID)
	if err != nil {
		// 回滚历史到本次输入之前的状态，避免残留的中间消息影响后续调用
		session.reactLoop.state.History = session.reactLoop.state.History[:historyLenBefore]
		return "", err
	}

	// 获取响应内容
	history := session.reactLoop.GetHistory()
	if len(history) > 0 {
		lastMsg := history[len(history)-1]
		if lastMsg.Role == llm.RoleAssistant && lastMsg.Content != "" {
			return lastMsg.Content, nil
		}
	}

	return "", nil
}

// RegisterTool 注册Tool
func (ge *GameEngine) RegisterTool(t tool.Tool, agents []string, category string) {
	ge.registry.Register(t, agents, category)
}

// Close 清理资源
func (ge *GameEngine) Close() error {
	// 目前无需清理
	return nil
}

// GameSession 游戏会话
type GameSession struct {
	ID            model.ID
	PlayerID      model.ID
	Engine        *GameEngine
	reactLoop     *ReActLoop
	combatSession *CombatSession // 战斗时非nil，接管ProcessInput
}

// GetID 获取会话ID
func (gs *GameSession) GetID() model.ID {
	return gs.ID
}

// SetPlayerID 设置玩家ID
func (gs *GameSession) SetPlayerID(playerID model.ID) {
	gs.PlayerID = playerID
}

// GetReactLoop 获取ReAct循环控制器
func (gs *GameSession) GetReactLoop() *ReActLoop {
	return gs.reactLoop
}
