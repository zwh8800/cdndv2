package gameengine

import (
	"context"
	"strings"
	"testing"

	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

type fakeCombatNarrativeLLM struct {
	gameID  model.ID
	sceneID model.ID
	pcID    model.ID
	enemyID model.ID
}

func (f *fakeCombatNarrativeLLM) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	toolNames := make(map[string]bool)
	for _, schema := range req.Tools {
		fn, _ := schema["function"].(map[string]any)
		name, _ := fn["name"].(string)
		toolNames[name] = true
	}

	if toolNames["delegate_task"] {
		lastUser := lastUserMessage(req.Messages)
		if !strings.Contains(lastUser, "哥布林") && !strings.Contains(lastUser, "战斗") {
			return textResponse("你检查战场，尘土渐渐落定，周围暂时安全。"), nil
		}
		if hasToolMessage(req.Messages) {
			return textResponse("战斗已经被战斗系统接管。"), nil
		}
		return &llm.CompletionResponse{
			Content: "前方的灌木后传来低吼，遭遇迅速升级为战斗。",
			ToolCalls: []llm.ToolCall{
				{
					ID:   "call_delegate_combat",
					Name: "delegate_task",
					Arguments: map[string]any{
						"agent_name": "combat_agent",
						"intent":     "使用已知参战者启动战斗",
					},
				},
			},
			FinishReason: llm.FinishReasonToolCalls,
		}, nil
	}

	if toolNames["combat_start"] {
		if hasToolMessage(req.Messages) {
			return textResponse("战斗初始化完成，交由独立战斗会话处理。"), nil
		}
		return &llm.CompletionResponse{
			ToolCalls: []llm.ToolCall{
				{
					ID:   "call_combat_start",
					Name: "combat_start",
					Arguments: map[string]any{
						"game_id":         string(f.gameID),
						"scene_id":        string(f.sceneID),
						"participant_ids": []any{string(f.pcID), string(f.enemyID)},
					},
				},
			},
			FinishReason: llm.FinishReasonToolCalls,
		}, nil
	}

	system := ""
	if len(req.Messages) > 0 {
		system = req.Messages[0].Content
	}
	switch {
	case strings.Contains(system, "战斗动作解析器"):
		return textResponse(`{"intent_type":"end_turn","confidence":1,"reason":"fake test"}`), nil
	case strings.Contains(system, "战斗叙事助手"):
		return textResponse("战斗态势更新，行动结果已经结算。"), nil
	case strings.Contains(system, "战斗记录员"):
		return textResponse("[战斗摘要]\n- 战斗已经结束，控制权回到探索叙事。\n- 后续可以检查战场、处理伤势或继续探索。"), nil
	case strings.Contains(system, "D&D战斗中的"):
		return textResponse("采取最直接的攻击行动。"), nil
	default:
		return textResponse("你检查战场，尘土渐渐落定，周围暂时安全。"), nil
	}
}

func (f *fakeCombatNarrativeLLM) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{Done: true}
	close(ch)
	return ch, nil
}

func textResponse(content string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Content:      content,
		FinishReason: llm.FinishReasonStop,
	}
}

func hasToolMessage(messages []llm.Message) bool {
	for _, msg := range messages {
		if msg.Role == llm.RoleTool {
			return true
		}
	}
	return false
}

func lastUserMessage(messages []llm.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == llm.RoleUser {
			return messages[i].Content
		}
	}
	return ""
}

func newFakeCombatNarrativeEngine(t *testing.T) (*GameEngine, *fakeCombatNarrativeLLM) {
	t.Helper()

	fake := &fakeCombatNarrativeLLM{}
	ge, err := NewGameEngine(EngineConfig{
		DNDEngineConfig: engine.Config{Storage: engine.DefaultConfig().Storage, DiceSeed: 1},
		OpenAIAPIKey:    "test-key",
		MaxIterations:   12,
		LogLevel:        "error",
	})
	if err != nil {
		t.Fatalf("NewGameEngine failed: %v", err)
	}
	ge.SetLLMClient(fake)
	return ge, fake
}

func prepareCombatNarrativeFixture(t *testing.T, ctx context.Context, ge *GameEngine, fake *fakeCombatNarrativeLLM) *GameSession {
	t.Helper()

	session, err := ge.NewGame(ctx, "Combat Narrative Test", "offline integration")
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	pcResult, err := ge.dndEngine.CreatePC(ctx, engine.CreatePCRequest{
		GameID: session.ID,
		PC: &engine.PlayerCharacterInput{
			Name:       "阿尔德里克",
			Race:       "人类",
			Class:      "战士",
			Background: "soldier",
			Level:      3,
			Alignment:  "lawful_good",
			AbilityScores: engine.AbilityScoresInput{
				Strength:     18,
				Dexterity:    14,
				Constitution: 16,
				Intelligence: 10,
				Wisdom:       12,
				Charisma:     8,
			},
			HitPoints: 35,
		},
	})
	if err != nil {
		t.Fatalf("CreatePC failed: %v", err)
	}
	session.SetPlayerID(pcResult.Actor.ID)

	scene, err := ge.dndEngine.CreateScene(ctx, engine.CreateSceneRequest{
		GameID:      session.ID,
		Name:        "北部森林",
		Description: "潮湿的林地里有低矮灌木和碎石。",
		SceneType:   model.SceneTypeOutdoor,
	})
	if err != nil {
		t.Fatalf("CreateScene failed: %v", err)
	}

	enemyResult, err := ge.dndEngine.CreateEnemy(ctx, engine.CreateEnemyRequest{
		GameID: session.ID,
		Enemy: &engine.EnemyInput{
			Name:            "哥布林",
			Description:     "一个握着弯刀的哥布林",
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
		t.Fatalf("CreateEnemy failed: %v", err)
	}

	if _, err := ge.dndEngine.SetPhase(ctx, session.ID, model.PhaseExploration, "fixture ready"); err != nil {
		t.Fatalf("SetPhase failed: %v", err)
	}
	if err := ge.dndEngine.SetCurrentScene(ctx, engine.SetCurrentSceneRequest{GameID: session.ID, SceneID: scene.Scene.ID}); err != nil {
		t.Fatalf("SetCurrentScene failed: %v", err)
	}

	fake.gameID = session.ID
	fake.sceneID = scene.Scene.ID
	fake.pcID = pcResult.Actor.ID
	fake.enemyID = enemyResult.Actor.ID
	return session
}

func TestProcessInputStartsCombatSessionAfterNarrativeTrigger(t *testing.T) {
	ctx := context.Background()
	ge, fake := newFakeCombatNarrativeEngine(t)
	defer ge.Close()
	session := prepareCombatNarrativeFixture(t, ctx, ge, fake)

	response, err := ge.ProcessInput(ctx, session, "我发现一只哥布林，拔出武器准备战斗")
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}
	if session.combatSession == nil {
		t.Fatal("expected CombatSession to be active after combat trigger")
	}
	phase, err := ge.dndEngine.GetPhase(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetPhase failed: %v", err)
	}
	if phase != model.PhaseCombat {
		t.Fatalf("expected phase combat, got %s", phase)
	}
	if !strings.Contains(response, "当前轮到") {
		t.Fatalf("expected combat turn handoff in response, got: %s", response)
	}
	if !strings.Contains(response, "动作") && !strings.Contains(response, "可用动作") {
		t.Fatalf("expected available action context in response, got: %s", response)
	}
}

func TestCombatEndInjectsSummaryAndReturnsToExploration(t *testing.T) {
	ctx := context.Background()
	ge, fake := newFakeCombatNarrativeEngine(t)
	defer ge.Close()
	session := prepareCombatNarrativeFixture(t, ctx, ge, fake)

	_, err := ge.dndEngine.StartCombat(ctx, engine.StartCombatRequest{
		GameID:         session.ID,
		SceneID:        fake.sceneID,
		ParticipantIDs: []model.ID{fake.pcID, fake.enemyID},
	})
	if err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}
	if _, err := ge.ProcessInput(ctx, session, ""); err != nil {
		t.Fatalf("initial combat ProcessInput failed: %v", err)
	}
	if session.combatSession == nil {
		t.Fatal("expected CombatSession to be active")
	}

	if err := ge.dndEngine.EndCombat(ctx, engine.EndCombatRequest{GameID: session.ID}); err != nil {
		t.Fatalf("EndCombat failed: %v", err)
	}
	if _, err := ge.ProcessInput(ctx, session, "继续"); err != nil {
		t.Fatalf("combat end ProcessInput failed: %v", err)
	}
	if session.combatSession != nil {
		t.Fatal("expected CombatSession to be released after combat end")
	}
	phase, err := ge.dndEngine.GetPhase(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetPhase failed: %v", err)
	}
	if phase != model.PhaseExploration {
		t.Fatalf("expected phase exploration, got %s", phase)
	}
	if !historyContains(session.reactLoop.GetHistory(), "[战斗摘要]") {
		t.Fatal("expected combat summary to be injected into main history")
	}
}

func TestNarrativeContinuesAfterCombatEnd(t *testing.T) {
	ctx := context.Background()
	ge, fake := newFakeCombatNarrativeEngine(t)
	defer ge.Close()
	session := prepareCombatNarrativeFixture(t, ctx, ge, fake)

	_, err := ge.dndEngine.StartCombat(ctx, engine.StartCombatRequest{
		GameID:         session.ID,
		SceneID:        fake.sceneID,
		ParticipantIDs: []model.ID{fake.pcID, fake.enemyID},
	})
	if err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}
	if _, err := ge.ProcessInput(ctx, session, ""); err != nil {
		t.Fatalf("initial combat ProcessInput failed: %v", err)
	}
	if err := ge.dndEngine.EndCombat(ctx, engine.EndCombatRequest{GameID: session.ID}); err != nil {
		t.Fatalf("EndCombat failed: %v", err)
	}
	if _, err := ge.ProcessInput(ctx, session, "继续"); err != nil {
		t.Fatalf("combat end ProcessInput failed: %v", err)
	}

	response, err := ge.ProcessInput(ctx, session, "我检查战场")
	if err != nil {
		t.Fatalf("post-combat ProcessInput failed: %v", err)
	}
	if session.combatSession != nil {
		t.Fatal("expected normal narrative path, got active CombatSession")
	}
	if !strings.Contains(response, "检查战场") {
		t.Fatalf("expected post-combat narrative response, got: %s", response)
	}
}

func historyContains(history []llm.Message, needle string) bool {
	for _, msg := range history {
		if strings.Contains(msg.Content, needle) {
			return true
		}
	}
	return false
}
