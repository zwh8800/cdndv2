package gameengine

import (
	"context"
	"fmt"
	"os"

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

	// 创建主Agent（包含子Agents）
	mainAgent := agent.NewMainAgent(registry, llmClient, subAgents)
	mainAgent.SetLogger(logger)

	// 创建ReAct循环
	maxIter := cfg.MaxIterations
	if maxIter == 0 {
		maxIter = 10
	}

	reactLoop := NewReActLoop(
		dndEngine,
		mainAgent,
		subAgents,
		registry,
		llmClient,
		maxIter,
	)
	reactLoop.SetLogger(logger)

	return &GameEngine{
		dndEngine: dndEngine,
		reactLoop: reactLoop,
		registry:  registry,
		mainAgent: mainAgent,
		llmClient: llmClient,
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
	// 添加玩家输入到历史
	session.reactLoop.state.History = append(
		session.reactLoop.state.History,
		llm.NewUserMessage(input),
	)

	// 执行ReAct循环
	err := session.reactLoop.Run(ctx, input, session.ID, session.PlayerID)
	if err != nil {
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
	ID        model.ID
	PlayerID  model.ID
	Engine    *GameEngine
	reactLoop *ReActLoop
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
