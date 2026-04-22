package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== create_player_character ==========

// CreatePlayerCharacterTool 一站式创建玩家角色
type CreatePlayerCharacterTool struct {
	EngineTool
}

func NewCreatePlayerCharacterTool(e *engine.Engine) *CreatePlayerCharacterTool {
	return &CreatePlayerCharacterTool{
		EngineTool: *NewEngineTool(
			"create_player_character",
			"一站式创建完整的玩家角色。自动验证种族/职业合法性，创建成功后若当前处于角色创建阶段则自动推进到探索阶段。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "角色名称",
					},
					"race": map[string]any{
						"type":        "string",
						"description": "种族，必须使用中文标准名",
						"enum":        []string{"人类", "精灵", "矮人", "半身人", "龙裔", "侏儒", "半精灵", "半兽人", "提夫林"},
					},
					"class": map[string]any{
						"type":        "string",
						"description": "职业，必须使用中文标准名",
						"enum":        []string{"野蛮人", "吟游诗人", "牧师", "德鲁伊", "战士", "武僧", "圣武士", "游侠", "游荡者", "术士", "邪术师", "法师"},
					},
					"ability_scores": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"strength":     map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
							"dexterity":    map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
							"constitution": map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
							"intelligence": map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
							"wisdom":       map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
							"charisma":     map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
						},
						"required":    []string{"strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma"},
						"description": "六项属性值",
					},
					"level": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"description": "角色等级（默认1）",
					},
					"background": map[string]any{
						"type":        "string",
						"description": "背景名称（中文标准名，可选）",
						"enum":        []string{"侍僧", "罪犯", "学者", "士兵"},
					},
					"alignment": map[string]any{
						"type":        "string",
						"description": "阵营（如 Lawful Good, True Neutral 等，可选）",
					},
				},
				"required": []string{"game_id", "name", "race", "class", "ability_scores"},
			},
			e,
			false,
		),
	}
}

func (t *CreatePlayerCharacterTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	name, err := RequireString(params, "name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	race, err := RequireString(params, "race")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	class, err := RequireString(params, "class")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	scores, err := RequireMap(params, "ability_scores")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	// 验证种族合法性
	_, raceErr := e.GetRace(ctx, engine.GetRaceRequest{Name: race})
	if raceErr != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效种族 '%s': %v", race, raceErr)}, nil
	}

	level := OptionalInt(params, "level", 1)
	alignment := OptionalString(params, "alignment", "True Neutral")
	background := OptionalString(params, "background", "")

	pc := &engine.PlayerCharacterInput{
		Name:      name,
		Race:      race,
		Class:     class,
		Level:     level,
		Alignment: alignment,
		AbilityScores: engine.AbilityScoresInput{
			Strength:     OptionalInt(scores, "strength", 10),
			Dexterity:    OptionalInt(scores, "dexterity", 10),
			Constitution: OptionalInt(scores, "constitution", 10),
			Intelligence: OptionalInt(scores, "intelligence", 10),
			Wisdom:       OptionalInt(scores, "wisdom", 10),
			Charisma:     OptionalInt(scores, "charisma", 10),
		},
		Background: background,
	}

	req := engine.CreatePCRequest{
		GameID: gameID,
		PC:     pc,
	}

	result, err := e.CreatePC(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	// 自动推进游戏阶段：若当前为 character_creation 则切换到 exploration
	phase, phaseErr := e.GetPhase(ctx, gameID)
	if phaseErr == nil && phase == model.PhaseCharacterCreation {
		_, _ = e.SetPhase(ctx, gameID, model.PhaseExploration, "角色创建完成，自动进入探索阶段")
	}

	return &ToolResult{
		Success: true,
		Data: map[string]any{
			"actor_id":    string(result.Actor.ID),
			"name":        result.Actor.Name,
			"hit_points":  result.Actor.HitPoints,
			"armor_class": result.Actor.ArmorClass,
		},
		Message: fmt.Sprintf("玩家角色 %s 创建成功（%s %s，等级%d）", name, race, class, level),
	}, nil
}

// ========== spawn_creature ==========

// SpawnCreatureTool 创建NPC/敌人/同伴的统一入口
type SpawnCreatureTool struct {
	EngineTool
}

func NewSpawnCreatureTool(e *engine.Engine) *SpawnCreatureTool {
	return &SpawnCreatureTool{
		EngineTool: *NewEngineTool(
			"spawn_creature",
			"创建NPC、敌人或同伴的统一入口。通过creature_type区分类型，支持从怪物模板快速创建敌人。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "生物名称",
					},
					"creature_type": map[string]any{
						"type":        "string",
						"enum":        []string{"npc", "enemy", "companion"},
						"description": "生物类型：npc=非玩家角色, enemy=敌人/怪物, companion=同伴/盟友",
					},
					"monster_template": map[string]any{
						"type":        "string",
						"description": "怪物模板名称（仅enemy类型，从怪物数据库查询属性自动填充）",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "描述（可选）",
					},
					"hit_points": map[string]any{
						"type":        "integer",
						"description": "生命值（可选，enemy类型）",
					},
					"armor_class": map[string]any{
						"type":        "integer",
						"description": "护甲等级（可选，enemy类型）",
					},
					"challenge_rating": map[string]any{
						"type":        "string",
						"description": "挑战等级（可选，enemy类型，如 1/4, 1, 5）",
					},
					"attitude": map[string]any{
						"type":        "string",
						"description": "对玩家的态度（仅npc类型）",
					},
					"occupation": map[string]any{
						"type":        "string",
						"description": "职业/身份（仅npc类型）",
					},
					"leader_id": map[string]any{
						"type":        "string",
						"description": "领导者（玩家）ID（仅companion类型）",
					},
				},
				"required": []string{"game_id", "name", "creature_type"},
			},
			e,
			false,
		),
	}
}

func (t *SpawnCreatureTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	name, err := RequireString(params, "name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	creatureType, err := RequireString(params, "creature_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	switch creatureType {
	case "npc":
		npc := &engine.NPCInput{
			Name:        name,
			Description: OptionalString(params, "description", ""),
			Occupation:  OptionalString(params, "occupation", ""),
			Attitude:    OptionalString(params, "attitude", ""),
		}
		req := engine.CreateNPCRequest{GameID: gameID, NPC: npc}
		result, err := e.CreateNPC(ctx, req)
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{
			Success: true,
			Data:    map[string]any{"actor_id": string(result.Actor.ID), "name": result.Actor.Name},
			Message: fmt.Sprintf("NPC %s 创建成功", name),
		}, nil

	case "enemy":
		enemy := &engine.EnemyInput{
			Name:            name,
			HitPoints:       OptionalInt(params, "hit_points", 0),
			ArmorClass:      OptionalInt(params, "armor_class", 0),
			ChallengeRating: OptionalString(params, "challenge_rating", ""),
		}
		req := engine.CreateEnemyRequest{GameID: gameID, Enemy: enemy}
		result, err := e.CreateEnemy(ctx, req)
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{
			Success: true,
			Data:    map[string]any{"actor_id": string(result.Actor.ID), "name": result.Actor.Name},
			Message: fmt.Sprintf("敌人 %s 创建成功", name),
		}, nil

	case "companion":
		companion := &engine.CompanionInput{
			Name:     name,
			LeaderID: OptionalString(params, "leader_id", ""),
			Loyalty:  OptionalInt(params, "loyalty", 0),
		}
		req := engine.CreateCompanionRequest{GameID: gameID, Companion: companion}
		result, err := e.CreateCompanion(ctx, req)
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{
			Success: true,
			Data:    map[string]any{"actor_id": string(result.Actor.ID), "name": result.Actor.Name},
			Message: fmt.Sprintf("同伴 %s 创建成功", name),
		}, nil

	default:
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("无效的creature_type: %s, 有效值: npc/enemy/companion", creatureType),
		}, nil
	}
}

// ========== query_character ==========

// QueryCharacterTool 统一角色查询
type QueryCharacterTool struct {
	EngineTool
}

func NewQueryCharacterTool(e *engine.Engine) *QueryCharacterTool {
	return &QueryCharacterTool{
		EngineTool: *NewEngineTool(
			"query_character",
			"统一角色信息查询。支持查询单个角色、角色列表、PC详情。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"query_type": map[string]any{
						"type":        "string",
						"enum":        []string{"single", "list", "pc_detail"},
						"description": "查询类型：single=单个角色, list=角色列表, pc_detail=玩家角色详情",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "角色ID（single和pc_detail时必填）",
					},
					"type_filter": map[string]any{
						"type":        "string",
						"description": "角色类型过滤（仅list时，可选：pc, npc, enemy, companion）",
					},
				},
				"required": []string{"game_id", "query_type"},
			},
			e,
			true,
		),
	}
}

func (t *QueryCharacterTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	queryType, err := RequireString(params, "query_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	switch queryType {
	case "single":
		actorIDStr, err := RequireString(params, "actor_id")
		if err != nil {
			return &ToolResult{Success: false, Error: "single查询需要actor_id参数"}, nil
		}
		result, err := e.GetActor(ctx, engine.GetActorRequest{
			GameID:  gameID,
			ActorID: model.ID(actorIDStr),
		})
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("角色: %s", result.Actor.Name)}, nil

	case "list":
		req := engine.ListActorsRequest{GameID: gameID}
		if tf := OptionalString(params, "type_filter", ""); tf != "" {
			req.Filter = &engine.ActorFilter{
				Types: []model.ActorType{model.ActorType(tf)},
			}
		}
		result, err := e.ListActors(ctx, req)
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个角色", len(result.Actors))}, nil

	case "pc_detail":
		actorIDStr, err := RequireString(params, "actor_id")
		if err != nil {
			return &ToolResult{Success: false, Error: "pc_detail查询需要actor_id参数"}, nil
		}
		result, err := e.GetPC(ctx, engine.GetPCRequest{
			GameID: gameID,
			PCID:   model.ID(actorIDStr),
		})
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("PC详情: %s", result.PC.Name)}, nil

	default:
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("无效的query_type: %s, 有效值: single/list/pc_detail", queryType),
		}, nil
	}
}
