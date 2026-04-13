package gameengine

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/zwh8800/dnd-core/pkg/engine"

	"github.com/zwh8800/cdndv2/game_engine/llm/openai"
)

func getOpenAIConfigFromEnv() openai.OpenAIConfig {
	config := openai.DefaultOpenAIConfig()

	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}

	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		config.Model = model
	}

	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	return config
}

func TestGameEngineFullFlow(t *testing.T) {
	ctx := context.Background()

	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   20,
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	session, err := gameEngine.NewGame(ctx, "Test Adventure", "A test adventure for testing")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	t.Logf("Created game with ID: %s", session.ID)

	playerInputs := []string{
		"你好，DM！我是一名人类战士，名叫Aldric。",
		"我的角色属性：力量16，敏捷12，体质14，智力10，感知10，魅力8。HP设为20。",
		"让我看看我的角色状态",
		"我进入酒馆，向酒保打听最近有什么新鲜事",
		"我向酒保询问是否有关于北部森林的传说",
		"我离开酒馆，前往北部森林探索",
		"我在森林中发现了一个可疑的山洞洞口",
		"我点燃火把，小心地进入山洞",
		"我查看山洞里有什么",
		"我发现了一只哥布林！拔出武器准备战斗",
		"我攻击哥布林",
	}

	for i, input := range playerInputs {
		t.Logf("\n=== Player Input %d: %s ===", i+1, input)

		response, err := gameEngine.ProcessInput(ctx, session, input)
		if err != nil {
			t.Logf("Error processing input: %v", err)
			continue
		}

		t.Logf("DM Response: %s", response)

		state := session.GetReactLoop().GetState()
		t.Logf("Phase: %v, Iteration: %d, History length: %d",
			state.CurrentPhase, state.Iteration, len(state.History))
	}

	t.Log("\n=== Final State ===")
	state := session.GetReactLoop().GetState()
	t.Logf("Final Phase: %v", state.CurrentPhase)
	t.Logf("Total Iterations: %d", state.Iteration)
	t.Logf("History messages: %d", len(state.History))

	for j, msg := range state.History {
		if j >= len(state.History)-5 {
			t.Logf("Message %d [%s]: %s", j, msg.Role, truncate(msg.Content, 100))
		}
	}
}

func TestGameEngineCombatFlow(t *testing.T) {
	ctx := context.Background()

	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   15,
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	session, err := gameEngine.NewGame(ctx, "Combat Test", "A test for combat mechanics")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	combatInputs := []string{
		"我是一名人类战士，力量18，体质16，敏捷14。HP 25。",
		"我在森林里遇到了一只狼，我先发起攻击",
		"我攻击狼",
		"狼咬了我一口，让我看看受伤情况",
		"我继续攻击狼",
		"狼倒下了，让我检查战利品",
	}

	for i, input := range combatInputs {
		t.Logf("\n=== Combat Input %d: %s ===", i+1, input)

		response, err := gameEngine.ProcessInput(ctx, session, input)
		if err != nil {
			t.Logf("Error: %v", err)
			continue
		}

		t.Logf("DM Response: %s", truncate(response, 200))
	}

	t.Log("\n=== Combat Test Complete ===")
}

func TestGameEngineExplorationFlow(t *testing.T) {
	ctx := context.Background()

	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   15,
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	session, err := gameEngine.NewGame(ctx, "Exploration Test", "A test for exploration")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	explorationInputs := []string{
		"我是一名精灵游侠，敏捷16，智力14，感知12。HP 18。",
		"我进入一个古老的废墟，试图寻找隐藏的宝藏",
		"我仔细搜索周围的区域",
		"我注意到墙上有些奇怪的符号，让我解读它们",
		"我找到了一条隐藏的通道，让我进入看看",
	}

	for i, input := range explorationInputs {
		t.Logf("\n=== Exploration Input %d: %s ===", i+1, input)

		response, err := gameEngine.ProcessInput(ctx, session, input)
		if err != nil {
			t.Logf("Error: %v", err)
			continue
		}

		t.Logf("DM Response: %s", truncate(response, 200))
	}
}

func TestGameEngineDialogueFlow(t *testing.T) {
	ctx := context.Background()

	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   15,
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	session, err := gameEngine.NewGame(ctx, "Dialogue Test", "A test for NPC dialogue")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	dialogueInputs := []string{
		"我是一个人类法师，智力18，魅力12，HP 14。",
		"我进入城镇的集市，看到一位神秘的老者在出售药水",
		"我向老者打招呼并询问他卖的是什么药水",
		"我询问有没有治疗药水，多少钱",
		"我购买了一瓶治疗药水",
		"我向老者询问附近是否有冒险任务",
	}

	for i, input := range dialogueInputs {
		t.Logf("\n=== Dialogue Input %d: %s ===", i+1, input)

		response, err := gameEngine.ProcessInput(ctx, session, input)
		if err != nil {
			t.Logf("Error: %v", err)
			continue
		}

		t.Logf("DM Response: %s", truncate(response, 200))
	}
}

func TestGameEngineLoadAndContinue(t *testing.T) {
	ctx := context.Background()

	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   10,
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	session, err := gameEngine.NewGame(ctx, "Save Test", "A test for save and load")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	gameID := session.ID

	_, err = gameEngine.ProcessInput(ctx, session, "我是一名战士，进入森林探险")
	if err != nil {
		t.Logf("First input error: %v", err)
	}

	t.Logf("Game saved, ID: %s", gameID)

	loadedSession, err := gameEngine.LoadGame(ctx, gameID)
	if err != nil {
		t.Fatalf("Failed to load game: %v", err)
	}

	t.Logf("Game loaded, ID: %s", loadedSession.ID)

	_, err = gameEngine.ProcessInput(ctx, loadedSession, "我继续在森林中探索")
	if err != nil {
		t.Logf("Continued input error: %v", err)
	}

	t.Log("Load and continue test completed")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func TestGameEngineDirectTools(t *testing.T) {
	ctx := context.Background()

	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   10,
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	session, err := gameEngine.NewGame(ctx, "Tools Test", "Testing direct tool usage")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	directToolInputs := []string{
		"我是一名战士，力量18，体质16，敏捷10，HP 30。",
		"让我使用roll_d20工具投一个D20",
		"让我投一个力量检定",
		"查看我的角色状态",
		"让我使用attack_roll工具做一次攻击投骰",
	}

	for i, input := range directToolInputs {
		t.Logf("\n=== Tool Test Input %d: %s ===", i+1, input)

		response, err := gameEngine.ProcessInput(ctx, session, input)
		if err != nil {
			t.Logf("Error: %v", err)
			continue
		}

		t.Logf("Response: %s", truncate(response, 200))
	}

	t.Log("Direct tools test completed")
}

func TestGameEngineInitialization(t *testing.T) {
	ctx := context.Background()

	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	t.Log("Creating game engine with configuration:")
	t.Logf("  - Model: %s", llmConfig.Model)
	t.Logf("  - MaxIterations: 20")

	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   20,
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	t.Log("Game engine created successfully")
	t.Logf("DND Engine: %v", gameEngine.GetDNDEngine() != nil)
	t.Logf("Tool Registry: %v", gameEngine.GetRegistry() != nil)
	t.Logf("Main Agent: %v", gameEngine.GetMainAgent() != nil)
	t.Logf("LLM Client: %v", gameEngine.GetLLMClient() != nil)

	dndEngine := gameEngine.GetDNDEngine()
	t.Logf("DND Engine type: %T", dndEngine)

	session, err := gameEngine.NewGame(ctx, "Init Test", "Testing initialization")
	if err != nil {
		t.Fatalf("Failed to create game session: %v", err)
	}

	t.Logf("Game session created with ID: %s", session.ID)
	t.Logf("Session type: %T", session)

	reactLoop := session.GetReactLoop()
	t.Logf("ReAct Loop initialized: %v", reactLoop != nil)

	state := reactLoop.GetState()
	t.Logf("Initial state - Phase: %v, Iteration: %d, History length: %d",
		state.CurrentPhase, state.Iteration, len(state.History))
}

func TestGameEngineMultipleSessions(t *testing.T) {
	ctx := context.Background()

	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   10,
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	session1, err := gameEngine.NewGame(ctx, "Session 1", "First session")
	if err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}

	session2, err := gameEngine.NewGame(ctx, "Session 2", "Second session")
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	t.Logf("Session 1 ID: %s", session1.ID)
	t.Logf("Session 2 ID: %s", session2.ID)

	_, err = gameEngine.ProcessInput(ctx, session1, "我是战士A，参加战斗")
	if err != nil {
		t.Logf("Session 1 input error: %v", err)
	}

	_, err = gameEngine.ProcessInput(ctx, session2, "我是法师B，施放魔法")
	if err != nil {
		t.Logf("Session 2 input error: %v", err)
	}

	t.Log("Multiple sessions test completed")
	fmt.Println("Test passed!")
}
