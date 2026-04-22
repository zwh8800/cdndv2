package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// =============================================================================
// 13. setup_scene - 一站式场景搭建
// =============================================================================

type SetupSceneTool struct {
	EngineTool
}

func NewSetupSceneTool(e *engine.Engine) *SetupSceneTool {
	return &SetupSceneTool{
		EngineTool: *NewEngineTool(
			"setup_scene",
			"一站式场景搭建：创建场景+设为当前+放置角色和物品+建立连接",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":     map[string]any{"type": "string", "description": "游戏会话ID"},
					"name":        map[string]any{"type": "string", "description": "场景名称"},
					"description": map[string]any{"type": "string", "description": "场景描述"},
					"scene_type": map[string]any{
						"type":        "string",
						"description": "场景类型（如 indoor/outdoor/dungeon/wilderness/urban/underground）",
					},
					"set_as_current": map[string]any{"type": "boolean", "description": "是否设为当前场景（默认true）"},
					"actor_ids": map[string]any{
						"type": "array", "items": map[string]any{"type": "string"},
						"description": "要放入场景的角色ID列表（可选）",
					},
					"item_ids": map[string]any{
						"type": "array", "items": map[string]any{"type": "string"},
						"description": "要放入场景的物品ID列表（可选）",
					},
					"connections": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"target_scene_id": map[string]any{"type": "string"},
								"description":     map[string]any{"type": "string"},
								"locked":          map[string]any{"type": "boolean"},
								"dc":              map[string]any{"type": "integer"},
								"hidden":          map[string]any{"type": "boolean"},
							},
							"required": []string{"target_scene_id", "description"},
						},
						"description": "场景连接配置（可选）",
					},
				},
				"required": []string{"game_id", "name", "description", "scene_type"},
			},
			e,
			false,
		),
	}
}

func (t *SetupSceneTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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
	description, err := RequireString(params, "description")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	sceneTypeStr, err := RequireString(params, "scene_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	// 创建场景
	createResult, createErr := e.CreateScene(ctx, engine.CreateSceneRequest{
		GameID: gameID, Name: name, Description: description, SceneType: model.SceneType(sceneTypeStr),
	})
	if createErr != nil {
		return &ToolResult{Success: false, Error: createErr.Error()}, nil
	}
	sceneID := createResult.Scene.ID

	// 设为当前场景
	if OptionalBool(params, "set_as_current", true) {
		_ = e.SetCurrentScene(ctx, engine.SetCurrentSceneRequest{GameID: gameID, SceneID: sceneID})
	}

	// 放置角色
	actorStrs := OptionalStringArray(params, "actor_ids")
	for _, aidStr := range actorStrs {
		_, _ = e.MoveActorToScene(ctx, engine.MoveActorToSceneRequest{
			GameID: gameID, ActorID: model.ID(aidStr), SceneID: sceneID,
		})
	}

	// 放置物品
	itemStrs := OptionalStringArray(params, "item_ids")
	for _, iidStr := range itemStrs {
		_ = e.AddItemToScene(ctx, engine.AddItemToSceneRequest{
			GameID: gameID, SceneID: sceneID, ItemID: model.ID(iidStr),
		})
	}

	// 建立连接
	if connList, ok := params["connections"].([]any); ok {
		for _, conn := range connList {
			connMap, ok := conn.(map[string]any)
			if !ok {
				continue
			}
			targetIDStr, _ := connMap["target_scene_id"].(string)
			connDesc, _ := connMap["description"].(string)
			if targetIDStr == "" || connDesc == "" {
				continue
			}
			_ = e.AddSceneConnection(ctx, engine.AddSceneConnectionRequest{
				GameID:        gameID,
				SceneID:       sceneID,
				TargetSceneID: model.ID(targetIDStr),
				Description:   connDesc,
				Locked:        OptionalBool(connMap, "locked", false),
				DC:            OptionalInt(connMap, "dc", 0),
				Hidden:        OptionalBool(connMap, "hidden", false),
			})
		}
	}

	return &ToolResult{
		Success: true,
		Data:    map[string]any{"scene_id": string(sceneID), "name": name},
		Message: fmt.Sprintf("场景 %s 搭建完成", name),
	}, nil
}

// =============================================================================
// 14. npc_social_interaction - NPC社交互动
// =============================================================================

type NPCSocialInteractionTool struct {
	EngineTool
}

func NewNPCSocialInteractionTool(e *engine.Engine) *NPCSocialInteractionTool {
	return &NPCSocialInteractionTool{
		EngineTool: *NewEngineTool(
			"npc_social_interaction",
			"与NPC社交互动。自动获取角色属性值进行社交检定",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":  map[string]any{"type": "string", "description": "游戏会话ID"},
					"actor_id": map[string]any{"type": "string", "description": "进行互动的PC角色ID"},
					"npc_id":   map[string]any{"type": "string", "description": "目标NPC的角色ID"},
					"check_type": map[string]any{
						"type": "string", "enum": []string{"persuasion", "intimidation", "deception", "performance"},
						"description": "社交检定类型",
					},
				},
				"required": []string{"game_id", "actor_id", "npc_id", "check_type"},
			},
			e,
			false,
		),
	}
}

func (t *NPCSocialInteractionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorID := model.ID(actorIDStr)

	npcIDStr, err := RequireString(params, "npc_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	npcID := model.ID(npcIDStr)

	checkTypeStr, err := RequireString(params, "check_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	// 自动获取PC详情（包含属性值）
	pcResult, pcerr := e.GetPC(ctx, engine.GetPCRequest{GameID: gameID, PCID: actorID})
	var ability int
	var profBonus int
	if pcerr == nil {
		ability = pcResult.PC.AbilityScores.Charisma
		profBonus = pcResult.PC.ProficiencyBonus
	} else {
		// 如果不是PC，使用默认值
		ability = 10
		profBonus = 2
	}

	interactResult, ierr := e.InteractWithNPC(ctx, engine.InteractWithNPCRequest{
		GameID:    gameID,
		NPCID:     npcID,
		CheckType: model.SocialCheckType(checkTypeStr),
		Ability:   ability,
		ProfBonus: profBonus,
		HasProf:   pcerr == nil,
	})
	if ierr != nil {
		return &ToolResult{Success: false, Error: ierr.Error()}, nil
	}

	// 获取NPC态度
	attitudeResult, _ := e.GetNPCAttitude(ctx, engine.GetNPCAttitudeRequest{GameID: gameID, NPCID: npcID})

	data := map[string]any{"interaction": interactResult}
	if attitudeResult != nil {
		data["attitude"] = attitudeResult.Attitude
		data["disposition"] = attitudeResult.Disposition
	}

	return &ToolResult{
		Success: true,
		Data:    data,
		Message: interactResult.Message,
	}, nil
}

// =============================================================================
// 15. manage_quest - 统一任务管理
// =============================================================================

type ManageQuestTool struct {
	EngineTool
}

func NewManageQuestTool(e *engine.Engine) *ManageQuestTool {
	return &ManageQuestTool{
		EngineTool: *NewEngineTool(
			"manage_quest",
			"统一任务管理：创建、接受、更新进度、完成或失败任务",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{"type": "string", "description": "游戏会话ID"},
					"operation": map[string]any{
						"type": "string", "enum": []string{"create", "accept", "update_objective", "complete", "fail"},
						"description": "任务操作类型",
					},
					"quest_id":          map[string]any{"type": "string", "description": "任务ID（accept/update/complete/fail时必需）"},
					"actor_id":          map[string]any{"type": "string", "description": "角色ID（accept时必需）"},
					"quest_name":        map[string]any{"type": "string", "description": "任务名称（create时必需）"},
					"quest_description": map[string]any{"type": "string", "description": "任务描述（create时必需）"},
					"giver_id":          map[string]any{"type": "string", "description": "任务发布者ID（create时必需）"},
					"giver_name":        map[string]any{"type": "string", "description": "任务发布者名称（create时必需）"},
					"objectives": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id":          map[string]any{"type": "string"},
								"description": map[string]any{"type": "string"},
								"required":    map[string]any{"type": "integer"},
							},
						},
						"description": "任务目标列表（create时）",
					},
					"rewards": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"experience": map[string]any{"type": "integer"},
							"gold":       map[string]any{"type": "integer"},
						},
						"description": "任务奖励（create时可选）",
					},
					"objective_id": map[string]any{"type": "string", "description": "目标ID（update_objective时必需）"},
					"progress":     map[string]any{"type": "integer", "description": "进度值（update_objective时必需）"},
				},
				"required": []string{"game_id", "operation"},
			},
			e,
			false,
		),
	}
}

func (t *ManageQuestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	op, err := RequireString(params, "operation")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	switch op {
	case "create":
		questName, nerr := RequireString(params, "quest_name")
		if nerr != nil {
			return &ToolResult{Success: false, Error: "create需要quest_name"}, nil
		}
		questDesc, derr := RequireString(params, "quest_description")
		if derr != nil {
			return &ToolResult{Success: false, Error: "create需要quest_description"}, nil
		}
		giverIDStr, gerr := RequireString(params, "giver_id")
		if gerr != nil {
			return &ToolResult{Success: false, Error: "create需要giver_id"}, nil
		}
		giverName, gnerr := RequireString(params, "giver_name")
		if gnerr != nil {
			return &ToolResult{Success: false, Error: "create需要giver_name"}, nil
		}

		objectives := make([]engine.ObjectiveInput, 0)
		if objList, ok := params["objectives"].([]any); ok {
			for _, obj := range objList {
				objMap, ok := obj.(map[string]any)
				if !ok {
					continue
				}
				objInput := engine.ObjectiveInput{
					ID:          OptionalString(objMap, "id", ""),
					Description: OptionalString(objMap, "description", ""),
				}
				if req, ok := objMap["required"].(float64); ok {
					objInput.Required = int(req)
				}
				objectives = append(objectives, objInput)
			}
		}

		var rewards *engine.QuestRewardsInput
		if r, ok := params["rewards"].(map[string]any); ok {
			rewards = &engine.QuestRewardsInput{}
			if exp, ok := r["experience"].(float64); ok {
				rewards.Experience = int(exp)
			}
			if gold, ok := r["gold"].(float64); ok {
				rewards.Gold = int(gold)
			}
		}

		result, cerr := e.CreateQuest(ctx, engine.CreateQuestRequest{
			GameID: gameID, Name: questName, Description: questDesc,
			GiverID: model.ID(giverIDStr), GiverName: giverName,
			Objectives: objectives, Rewards: rewards,
		})
		if cerr != nil {
			return &ToolResult{Success: false, Error: cerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	case "accept":
		questIDStr, qerr := RequireString(params, "quest_id")
		if qerr != nil {
			return &ToolResult{Success: false, Error: "accept需要quest_id"}, nil
		}
		actorIDStr, aerr := RequireString(params, "actor_id")
		if aerr != nil {
			return &ToolResult{Success: false, Error: "accept需要actor_id"}, nil
		}
		result, aerr2 := e.AcceptQuest(ctx, engine.AcceptQuestRequest{
			GameID: gameID, QuestID: model.ID(questIDStr), ActorID: model.ID(actorIDStr),
		})
		if aerr2 != nil {
			return &ToolResult{Success: false, Error: aerr2.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	case "update_objective":
		questIDStr, qerr := RequireString(params, "quest_id")
		if qerr != nil {
			return &ToolResult{Success: false, Error: "update_objective需要quest_id"}, nil
		}
		objID, oerr := RequireString(params, "objective_id")
		if oerr != nil {
			return &ToolResult{Success: false, Error: "update_objective需要objective_id"}, nil
		}
		progress, perr := RequireInt(params, "progress")
		if perr != nil {
			return &ToolResult{Success: false, Error: "update_objective需要progress"}, nil
		}
		result, uerr := e.UpdateQuestObjective(ctx, engine.UpdateQuestObjectiveRequest{
			GameID: gameID, QuestID: model.ID(questIDStr), ObjectiveID: objID, Progress: progress,
		})
		if uerr != nil {
			return &ToolResult{Success: false, Error: uerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	case "complete":
		questIDStr, qerr := RequireString(params, "quest_id")
		if qerr != nil {
			return &ToolResult{Success: false, Error: "complete需要quest_id"}, nil
		}
		result, cerr := e.CompleteQuest(ctx, engine.CompleteQuestRequest{GameID: gameID, QuestID: model.ID(questIDStr)})
		if cerr != nil {
			return &ToolResult{Success: false, Error: cerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	case "fail":
		questIDStr, qerr := RequireString(params, "quest_id")
		if qerr != nil {
			return &ToolResult{Success: false, Error: "fail需要quest_id"}, nil
		}
		result, ferr := e.FailQuest(ctx, engine.FailQuestRequest{GameID: gameID, QuestID: model.ID(questIDStr)})
		if ferr != nil {
			return &ToolResult{Success: false, Error: ferr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: result.Message}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效操作: %s", op)}, nil
	}
}

// =============================================================================
// 16. travel - 完整旅行流程
// =============================================================================

type TravelTool struct {
	EngineTool
}

func NewTravelTool(e *engine.Engine) *TravelTool {
	return &TravelTool{
		EngineTool: *NewEngineTool(
			"travel",
			"发起并推进旅行。包含旅行开始、推进和遭遇检定",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id":     map[string]any{"type": "string", "description": "游戏会话ID"},
					"destination": map[string]any{"type": "string", "description": "目的地名称"},
					"terrain": map[string]any{
						"type":        "string",
						"description": "地形类型（如 normal/difficult/forest/mountain/desert/swamp/arctic/coastal）",
					},
					"distance": map[string]any{"type": "number", "description": "距离（英里）"},
					"pace": map[string]any{
						"type": "string", "enum": []string{"slow", "normal", "fast"},
						"description": "行进速度（默认normal）",
					},
					"hours_to_advance":  map[string]any{"type": "integer", "description": "推进旅行的小时数（可选，提供则自动推进）"},
					"check_encounters": map[string]any{"type": "boolean", "description": "是否进行遭遇检定（默认true）"},
				},
				"required": []string{"game_id", "destination", "terrain", "distance"},
			},
			e,
			false,
		),
	}
}

func (t *TravelTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	destination, err := RequireString(params, "destination")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	terrainStr, err := RequireString(params, "terrain")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	distance, err := RequireFloat(params, "distance")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	pace := model.TravelPace(OptionalString(params, "pace", "normal"))

	// 开始旅行
	startResult, serr := e.StartTravel(ctx, engine.StartTravelRequest{
		GameID: gameID, Destination: destination, Pace: pace,
		Terrain: model.TerrainType(terrainStr), Distance: distance,
	})
	if serr != nil {
		return &ToolResult{Success: false, Error: serr.Error()}, nil
	}

	data := map[string]any{"travel_start": startResult}

	// 推进旅行
	hours := OptionalInt(params, "hours_to_advance", 0)
	if hours > 0 {
		advResult, aerr := e.AdvanceTravel(ctx, engine.AdvanceTravelRequest{GameID: gameID, Hours: hours})
		if aerr != nil {
			return &ToolResult{Success: false, Error: aerr.Error()}, nil
		}
		data["travel_advance"] = advResult
	}

	// 遭遇检定
	if OptionalBool(params, "check_encounters", true) {
		encResult, eerr := e.PerformEncounterCheck(ctx, engine.PerformEncounterCheckRequest{GameID: gameID})
		if eerr == nil {
			data["encounter_check"] = encResult
			if encResult.Encountered {
				data["encounter_warning"] = "遭遇随机事件！"
			}
		}
	}

	return &ToolResult{
		Success: true,
		Data:    data,
		Message: startResult.Message,
	}, nil
}

// =============================================================================
// 17. query_world - 统一世界信息查询
// =============================================================================

type QueryWorldTool struct {
	EngineTool
}

func NewQueryWorldTool(e *engine.Engine) *QueryWorldTool {
	return &QueryWorldTool{
		EngineTool: *NewEngineTool(
			"query_world",
			"统一世界信息查询：场景、NPC态度、任务等",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{"type": "string", "description": "游戏会话ID"},
					"query_type": map[string]any{
						"type": "string",
						"enum": []string{"current_scene", "scene_detail", "scene_list", "scene_actors", "scene_items", "npc_attitude", "quest_list", "quest_detail", "actor_quests"},
						"description": "查询类型",
					},
					"scene_id": map[string]any{"type": "string", "description": "场景ID（scene_detail/scene_actors/scene_items时必需）"},
					"npc_id":   map[string]any{"type": "string", "description": "NPC ID（npc_attitude时必需）"},
					"quest_id": map[string]any{"type": "string", "description": "任务ID（quest_detail时必需）"},
					"actor_id": map[string]any{"type": "string", "description": "角色ID（actor_quests时必需）"},
					"status":   map[string]any{"type": "string", "description": "任务状态过滤（quest_list/actor_quests时可选）"},
				},
				"required": []string{"game_id", "query_type"},
			},
			e,
			true, // 只读
		),
	}
}

func (t *QueryWorldTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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
	case "current_scene":
		result, qerr := e.GetCurrentScene(ctx, engine.GetCurrentSceneRequest{GameID: gameID})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("当前场景: %s", result.Name)}, nil

	case "scene_detail":
		sceneIDStr, serr := RequireString(params, "scene_id")
		if serr != nil {
			return &ToolResult{Success: false, Error: "scene_detail需要scene_id"}, nil
		}
		result, qerr := e.GetScene(ctx, engine.GetSceneRequest{GameID: gameID, SceneID: model.ID(sceneIDStr)})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("场景: %s", result.Name)}, nil

	case "scene_list":
		result, qerr := e.ListScenes(ctx, engine.ListScenesRequest{GameID: gameID})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个场景", len(result.Scenes))}, nil

	case "scene_actors":
		sceneIDStr, serr := RequireString(params, "scene_id")
		if serr != nil {
			return &ToolResult{Success: false, Error: "scene_actors需要scene_id"}, nil
		}
		result, qerr := e.GetSceneActors(ctx, engine.GetSceneActorsRequest{GameID: gameID, SceneID: model.ID(sceneIDStr)})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("场景中有 %d 个角色", len(result.Actors))}, nil

	case "scene_items":
		sceneIDStr, serr := RequireString(params, "scene_id")
		if serr != nil {
			return &ToolResult{Success: false, Error: "scene_items需要scene_id"}, nil
		}
		result, qerr := e.GetSceneItems(ctx, engine.GetSceneItemsRequest{GameID: gameID, SceneID: model.ID(sceneIDStr)})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("场景中有 %d 件物品", len(result.Items))}, nil

	case "npc_attitude":
		npcIDStr, nerr := RequireString(params, "npc_id")
		if nerr != nil {
			return &ToolResult{Success: false, Error: "npc_attitude需要npc_id"}, nil
		}
		result, qerr := e.GetNPCAttitude(ctx, engine.GetNPCAttitudeRequest{GameID: gameID, NPCID: model.ID(npcIDStr)})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("NPC态度: %s", result.Attitude)}, nil

	case "quest_list":
		req := engine.ListQuestsRequest{GameID: gameID}
		if status := OptionalString(params, "status", ""); status != "" {
			s := model.QuestStatus(status)
			req.Status = &s
		}
		result, qerr := e.ListQuests(ctx, req)
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个任务", len(result.Quests))}, nil

	case "quest_detail":
		questIDStr, qerr := RequireString(params, "quest_id")
		if qerr != nil {
			return &ToolResult{Success: false, Error: "quest_detail需要quest_id"}, nil
		}
		result, qerr2 := e.GetQuest(ctx, engine.GetQuestRequest{GameID: gameID, QuestID: model.ID(questIDStr)})
		if qerr2 != nil {
			return &ToolResult{Success: false, Error: qerr2.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("任务: %s (%s)", result.Name, result.Status)}, nil

	case "actor_quests":
		actorIDStr, aerr := RequireString(params, "actor_id")
		if aerr != nil {
			return &ToolResult{Success: false, Error: "actor_quests需要actor_id"}, nil
		}
		req := engine.GetActorQuestsRequest{GameID: gameID, ActorID: model.ID(actorIDStr)}
		if status := OptionalString(params, "status", ""); status != "" {
			s := model.QuestStatus(status)
			req.Status = &s
		}
		result, qerr := e.GetActorQuests(ctx, req)
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("角色有 %d 个任务", len(result.Quests))}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效的query_type: %s", queryType)}, nil
	}
}
