package gameengine

import (
	"context"
	"testing"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// TestCombatSessionFullBattle 模拟一场完整的 D&D 战斗。
// 直接测试 CombatSession（绕过 GameEngine.ProcessInput），验证：
//   - 战斗初始化（LLM 调用 next_turn_with_actions）
//   - 玩家回合处理（LLM 匹配动作并执行）
//   - 敌人回合自动处理（EnemyAIAgent 生成意图）
//   - 战斗结束检测
//
// 运行: OPENAI_API_KEY=sk-... go test -run TestCombatSessionFullBattle -v -timeout 10m ./game_engine/
func TestCombatSessionFullBattle(t *testing.T) {
	ctx := context.Background()

	// === Guard: 需要真实 LLM ===
	llmConfig := getOpenAIConfigFromEnv()
	if llmConfig.APIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	// === 创建 GameEngine ===
	gameEngine, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   20,
		LogLevel:        "debug",
	})
	if err != nil {
		t.Fatalf("Failed to create game engine: %v", err)
	}
	defer gameEngine.Close()

	dndEngine := gameEngine.GetDNDEngine()

	// === 数据准备（纯 dnd-core API，无 LLM 调用）===

	// 1. 创建游戏
	gameResult, err := dndEngine.NewGame(ctx, engine.NewGameRequest{
		Name:        "Combat Session Test",
		Description: "Integration test for CombatSession full battle",
	})
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}
	gameID := gameResult.Game.ID
	t.Logf("Game created: %s", gameID)

	// 2. 创建玩家角色（5级战士，强力属性，确保快速击杀弱敌）
	pcResult, err := dndEngine.CreatePC(ctx, engine.CreatePCRequest{
		GameID: gameID,
		PC: &engine.PlayerCharacterInput{
			Name:       "阿尔德里克",
			Race:       "人类",
			Class:      "战士",
			Background: "soldier",
			Level:      5,
			Alignment:  "lawful_good",
			AbilityScores: engine.AbilityScoresInput{
				Strength:     18,
				Dexterity:    14,
				Constitution: 16,
				Intelligence: 10,
				Wisdom:       12,
				Charisma:     8,
			},
			HitPoints: 44,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create PC: %v", err)
	}
	pcID := pcResult.Actor.ID
	t.Logf("PC created: %s (阿尔德里克, HP %d/%d, AC %d)", pcID,
		pcResult.Actor.HitPoints.Current, pcResult.Actor.HitPoints.Maximum, pcResult.Actor.ArmorClass)

	// 3. 创建敌人（极弱哥布林，7HP AC10，1-2回合可击杀）
	enemyResult, err := dndEngine.CreateEnemy(ctx, engine.CreateEnemyRequest{
		GameID: gameID,
		Enemy: &engine.EnemyInput{
			Name:            "哥布林",
			Description:     "一个虚弱的哥布林",
			Size:            "Small",
			CreatureType:    "Humanoid",
			Speed:           30,
			ChallengeRating: "1/4",
			HitPoints:       7,
			ArmorClass:      10,
			AttackBonus:     2,
			DamagePerRound:  3,
			XPValue:         25,
			AbilityScores: engine.AbilityScoresInput{
				Strength:     8,
				Dexterity:    10,
				Constitution: 8,
				Intelligence: 6,
				Wisdom:       8,
				Charisma:     6,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create enemy: %v", err)
	}
	enemyID := enemyResult.Actor.ID
	t.Logf("Enemy created: %s (哥布林, HP %d/%d, AC %d)", enemyID,
		enemyResult.Actor.HitPoints.Current, enemyResult.Actor.HitPoints.Maximum, enemyResult.Actor.ArmorClass)

	// 4. 切换到探索阶段（SetupCombat 要求 PhaseExploration）
	_, err = dndEngine.SetPhase(ctx, gameID, model.PhaseExploration, "准备战斗")
	if err != nil {
		t.Fatalf("Failed to set phase: %v", err)
	}
	t.Log("Phase set to exploration")

	// 5. 初始化战斗（掷先攻、设置回合顺序）
	setupResult, err := dndEngine.SetupCombat(ctx, engine.SetupCombatRequest{
		GameID:         gameID,
		ParticipantIDs: []model.ID{pcID, enemyID},
	})
	if err != nil {
		t.Fatalf("Failed to setup combat: %v", err)
	}
	t.Logf("Combat setup complete. First turn: %s (%s), Round %d",
		setupResult.FirstTurn.ActorName, setupResult.FirstTurn.ActorType, setupResult.FirstTurn.Round)

	// === 创建 CombatSession 并运行战斗 ===

	llmClient := gameEngine.GetLLMClient()
	registry := gameEngine.GetRegistry()

	cs := NewCombatSession(gameID, pcID, llmClient, registry, dndEngine, gameEngine.enemyAI, gameEngine.logger)

	// 初始化战斗（LLM 调用 next_turn_with_actions，生成开场叙述）
	t.Log("\n=== Combat: Initialize ===")
	result, err := cs.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize combat session: %v", err)
	}
	if result.Response == "" {
		t.Fatal("Initialize returned empty response")
	}
	t.Logf("Response:\n%s", truncate(result.Response, 500))

	if result.CombatEnded {
		t.Log("Combat ended during initialization")
		return
	}

	// 战斗循环：玩家重复攻击直到战斗结束
	const maxRounds = 10
	playerAction := "我用武器攻击哥布林，然后结束回合"

	for round := 1; round <= maxRounds; round++ {
		t.Logf("\n=== Combat: Turn %d ===", round)
		t.Logf("Player input: %s", playerAction)

		result, err = cs.ProcessInput(ctx, playerAction)
		if err != nil {
			t.Fatalf("ProcessInput failed at turn %d: %v", round, err)
		}
		if result.Response == "" {
			t.Fatalf("ProcessInput returned empty response at turn %d", round)
		}
		t.Logf("Response:\n%s", truncate(result.Response, 500))

		if result.CombatEnded {
			t.Logf("\n=== Combat Ended at Turn %d ===", round)
			break
		}
	}

	// === 结果记录 ===
	if result.CombatEnded {
		t.Log("Combat completed successfully")
	} else {
		t.Logf("WARNING: Combat did not end within %d turns (LLM/dice non-determinism)", maxRounds)
	}
}
