package game_summary

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// GameSummary 游戏状态摘要
type GameSummary struct {
	// 游戏基本信息
	GameID   model.ID `json:"game_id"`
	GameName string   `json:"game_name"`
	Phase    string   `json:"phase"`

	// 当前场景
	CurrentScene *SceneSummary `json:"current_scene,omitempty"`

	// 玩家角色
	Player *ActorSummary `json:"player,omitempty"`

	// 战斗状态
	Combat *CombatSummary `json:"combat,omitempty"`

	// 任务状态
	ActiveQuests []QuestSummary `json:"active_quests,omitempty"`

	// 扩展信息
	PlayerInput      string `json:"player_input,omitempty"`
	LastActionResult string `json:"last_action_result,omitempty"`
}

// SceneSummary 场景摘要
type SceneSummary struct {
	ID          model.ID `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
}

// ActorSummary 角色摘要
type ActorSummary struct {
	ID         model.ID `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	HitPoints  int      `json:"hit_points"`
	MaxHP      int      `json:"max_hp"`
	ArmorClass int      `json:"armor_class"`
}

// CombatSummary 战斗摘要
type CombatSummary struct {
	Status        string `json:"status"`
	Round         int    `json:"round"`
	TurnActorID   string `json:"turn_actor_id"`
	TurnActorName string `json:"turn_actor_name"`
}

// QuestSummary 任务摘要
type QuestSummary struct {
	ID          model.ID `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
}

// CollectSummary 收集游戏状态摘要
func CollectSummary(ctx context.Context, e *engine.Engine, gameID model.ID, playerID model.ID) (*GameSummary, error) {
	summary := &GameSummary{
		GameID: gameID,
	}

	// 获取当前场景
	sceneInfo, err := e.GetCurrentScene(ctx, engine.GetCurrentSceneRequest{
		GameID: gameID,
	})
	if err == nil && sceneInfo != nil {
		summary.CurrentScene = &SceneSummary{
			ID:          sceneInfo.ID,
			Name:        sceneInfo.Name,
			Description: sceneInfo.Description,
		}
	}

	// 获取玩家角色
	if playerID != "" {
		pcResult, err := e.GetPC(ctx, engine.GetPCRequest{
			GameID: gameID,
			PCID:   playerID,
		})
		if err == nil && pcResult.PC != nil {
			summary.Player = &ActorSummary{
				ID:         pcResult.PC.ID,
				Name:       pcResult.PC.Name,
				Type:       "PC",
				HitPoints:  pcResult.PC.HitPoints.Current,
				MaxHP:      pcResult.PC.HitPoints.Maximum,
				ArmorClass: pcResult.PC.ArmorClass,
			}
		}
	}

	// 获取战斗状态
	combatResult, err := e.GetCurrentCombat(ctx, engine.GetCurrentCombatRequest{
		GameID: gameID,
	})
	if err == nil && combatResult != nil && combatResult.Combat != nil {
		status := string(combatResult.Combat.Status)
		summary.Combat = &CombatSummary{
			Status: status,
			Round:  combatResult.Combat.Round,
		}
		if combatResult.Combat.CurrentTurn != nil {
			summary.Combat.TurnActorID = string(combatResult.Combat.CurrentTurn.ActorID)
			summary.Combat.TurnActorName = combatResult.Combat.CurrentTurn.ActorName
		}
	}

	// 获取活跃任务
	questsResult, err := e.GetActorQuests(ctx, engine.GetActorQuestsRequest{
		GameID:  gameID,
		ActorID: playerID,
	})
	if err == nil && questsResult != nil {
		for _, quest := range questsResult.Quests {
			if quest.Status == model.QuestStatusActive {
				summary.ActiveQuests = append(summary.ActiveQuests, QuestSummary{
					ID:          quest.ID,
					Title:       quest.Name,
					Description: quest.Description,
					Status:      string(quest.Status),
				})
			}
		}
	}

	// 获取当前 Phase
	phaseResult, err := e.GetPhase(ctx, gameID)
	if err == nil {
		summary.Phase = string(phaseResult)
	}

	return summary, nil
}

// NewGameSummary 创建新的游戏摘要
func NewGameSummary(gameID model.ID) *GameSummary {
	return &GameSummary{
		GameID: gameID,
	}
}

// UpdatePlayerInput 更新玩家输入
func (s *GameSummary) UpdatePlayerInput(input string) {
	s.PlayerInput = input
}

// UpdateLastActionResult 更新上次执行结果
func (s *GameSummary) UpdateLastActionResult(result string) {
	s.LastActionResult = result
}

// String 返回摘要的字符串表示
func (s *GameSummary) String() string {
	result := fmt.Sprintf("Game: %s (ID: %s)\n", s.GameName, s.GameID)
	result += fmt.Sprintf("Phase: %s\n", s.Phase)

	if s.CurrentScene != nil {
		result += fmt.Sprintf("Scene: %s\n", s.CurrentScene.Name)
	}

	if s.Player != nil {
		result += fmt.Sprintf("Player: %s (HP: %d/%d, AC: %d)\n",
			s.Player.Name, s.Player.HitPoints, s.Player.MaxHP, s.Player.ArmorClass)
	}

	if s.Combat != nil {
		result += fmt.Sprintf("Combat: Round %d, Turn: %s\n", s.Combat.Round, s.Combat.TurnActorName)
	}

	if len(s.ActiveQuests) > 0 {
		result += "Active Quests:\n"
		for _, q := range s.ActiveQuests {
			result += fmt.Sprintf("  - %s: %s\n", q.Title, q.Description)
		}
	}

	return result
}
