package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 任务管理工具 ==========

// CreateQuestTool 创建任务
type CreateQuestTool struct {
	EngineTool
}

func NewCreateQuestTool(e *engine.Engine) *CreateQuestTool {
	return &CreateQuestTool{
		EngineTool: *NewEngineTool(
			"create_quest",
			"创建新的游戏任务",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "任务名称",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "任务描述",
					},
					"giver_id": map[string]any{
						"type":        "string",
						"description": "任务发布者ID",
					},
					"giver_name": map[string]any{
						"type":        "string",
						"description": "任务发布者名称",
					},
					"objectives": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id":          map[string]any{"type": "string", "description": "目标ID"},
								"description": map[string]any{"type": "string", "description": "目标描述"},
								"required":    map[string]any{"type": "integer", "description": "完成所需数量"},
							},
							"required": []string{"id", "description"},
						},
						"description": "任务目标列表",
					},
					"rewards": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"experience": map[string]any{"type": "integer", "description": "经验奖励"},
							"gold":       map[string]any{"type": "integer", "description": "金币奖励"},
						},
						"description": "任务奖励（可选）",
					},
				},
				"required": []string{"game_id", "name", "description", "giver_id", "giver_name", "objectives"},
			},
			e,
			false,
		),
	}
}

func (t *CreateQuestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	giverIDStr, err := RequireString(params, "giver_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	giverID := model.ID(giverIDStr)

	giverName, err := RequireString(params, "giver_name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	objectives := make([]engine.ObjectiveInput, 0)
	if objList, ok := params["objectives"].([]any); ok {
		for _, obj := range objList {
			objMap, ok := obj.(map[string]any)
			if !ok {
				continue
			}
			objInput := engine.ObjectiveInput{
				ID:          objMap["id"].(string),
				Description: objMap["description"].(string),
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

	req := engine.CreateQuestRequest{
		GameID:      gameID,
		Name:        name,
		Description: description,
		GiverID:     giverID,
		GiverName:   giverName,
		Objectives:  objectives,
		Rewards:     rewards,
	}

	result, err := e.CreateQuest(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.Message,
	}, nil
}

// GetQuestTool 获取任务信息
type GetQuestTool struct {
	EngineTool
}

func NewGetQuestTool(e *engine.Engine) *GetQuestTool {
	return &GetQuestTool{
		EngineTool: *NewEngineTool(
			"get_quest",
			"获取指定任务的详细信息",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"quest_id": map[string]any{
						"type":        "string",
						"description": "任务ID",
					},
				},
				"required": []string{"game_id", "quest_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetQuestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	questIDStr, err := RequireString(params, "quest_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	questID := model.ID(questIDStr)

	req := engine.GetQuestRequest{
		GameID:  gameID,
		QuestID: questID,
	}

	result, err := e.GetQuest(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("任务: %s (%s)", result.Name, result.Status),
	}, nil
}

// ListQuestsTool 列出所有任务
type ListQuestsTool struct {
	EngineTool
}

func NewListQuestsTool(e *engine.Engine) *ListQuestsTool {
	return &ListQuestsTool{
		EngineTool: *NewEngineTool(
			"list_quests",
			"列出游戏中的所有任务，可按状态过滤",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"status": map[string]any{
						"type":        "string",
						"description": "按状态过滤 (available=可接, active=进行中, completed=已完成, failed=已失败)",
					},
				},
				"required": []string{"game_id"},
			},
			e,
			true,
		),
	}
}

func (t *ListQuestsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	req := engine.ListQuestsRequest{
		GameID: gameID,
	}

	if status := OptionalString(params, "status", ""); status != "" {
		s := model.QuestStatus(status)
		req.Status = &s
	}

	result, err := e.ListQuests(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个任务", len(result.Quests)),
	}, nil
}

// AcceptQuestTool 接受任务
type AcceptQuestTool struct {
	EngineTool
}

func NewAcceptQuestTool(e *engine.Engine) *AcceptQuestTool {
	return &AcceptQuestTool{
		EngineTool: *NewEngineTool(
			"accept_quest",
			"角色接受指定任务",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"quest_id": map[string]any{
						"type":        "string",
						"description": "任务ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "接受任务的角色ID",
					},
				},
				"required": []string{"game_id", "quest_id", "actor_id"},
			},
			e,
			false,
		),
	}
}

func (t *AcceptQuestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	questIDStr, err := RequireString(params, "quest_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	questID := model.ID(questIDStr)

	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorID := model.ID(actorIDStr)

	req := engine.AcceptQuestRequest{
		GameID:  gameID,
		QuestID: questID,
		ActorID: actorID,
	}

	result, err := e.AcceptQuest(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.Message,
	}, nil
}

// UpdateQuestObjectiveTool 更新任务目标进度
type UpdateQuestObjectiveTool struct {
	EngineTool
}

func NewUpdateQuestObjectiveTool(e *engine.Engine) *UpdateQuestObjectiveTool {
	return &UpdateQuestObjectiveTool{
		EngineTool: *NewEngineTool(
			"update_quest_objective",
			"更新任务目标的进度",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"quest_id": map[string]any{
						"type":        "string",
						"description": "任务ID",
					},
					"objective_id": map[string]any{
						"type":        "string",
						"description": "目标ID",
					},
					"progress": map[string]any{
						"type":        "integer",
						"description": "进度增量",
					},
				},
				"required": []string{"game_id", "quest_id", "objective_id", "progress"},
			},
			e,
			false,
		),
	}
}

func (t *UpdateQuestObjectiveTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	questIDStr, err := RequireString(params, "quest_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	questID := model.ID(questIDStr)

	objectiveID, err := RequireString(params, "objective_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	progress, err := RequireInt(params, "progress")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.UpdateQuestObjectiveRequest{
		GameID:      gameID,
		QuestID:     questID,
		ObjectiveID: objectiveID,
		Progress:    progress,
	}

	result, err := e.UpdateQuestObjective(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.Message,
	}, nil
}

// CompleteQuestTool 完成任务
type CompleteQuestTool struct {
	EngineTool
}

func NewCompleteQuestTool(e *engine.Engine) *CompleteQuestTool {
	return &CompleteQuestTool{
		EngineTool: *NewEngineTool(
			"complete_quest",
			"完成任务并发放奖励",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"quest_id": map[string]any{
						"type":        "string",
						"description": "任务ID",
					},
				},
				"required": []string{"game_id", "quest_id"},
			},
			e,
			false,
		),
	}
}

func (t *CompleteQuestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	questIDStr, err := RequireString(params, "quest_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	questID := model.ID(questIDStr)

	req := engine.CompleteQuestRequest{
		GameID:  gameID,
		QuestID: questID,
	}

	result, err := e.CompleteQuest(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.Message,
	}, nil
}

// FailQuestTool 任务失败
type FailQuestTool struct {
	EngineTool
}

func NewFailQuestTool(e *engine.Engine) *FailQuestTool {
	return &FailQuestTool{
		EngineTool: *NewEngineTool(
			"fail_quest",
			"标记任务为失败状态",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"quest_id": map[string]any{
						"type":        "string",
						"description": "任务ID",
					},
				},
				"required": []string{"game_id", "quest_id"},
			},
			e,
			false,
		),
	}
}

func (t *FailQuestTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	questIDStr, err := RequireString(params, "quest_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	questID := model.ID(questIDStr)

	req := engine.FailQuestRequest{
		GameID:  gameID,
		QuestID: questID,
	}

	result, err := e.FailQuest(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.Message,
	}, nil
}

// GetActorQuestsTool 获取角色任务
type GetActorQuestsTool struct {
	EngineTool
}

func NewGetActorQuestsTool(e *engine.Engine) *GetActorQuestsTool {
	return &GetActorQuestsTool{
		EngineTool: *NewEngineTool(
			"get_actor_quests",
			"获取角色接受的所有任务",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"actor_id": map[string]any{
						"type":        "string",
						"description": "角色ID",
					},
					"status": map[string]any{
						"type":        "string",
						"description": "按状态过滤（可选）",
					},
				},
				"required": []string{"game_id", "actor_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetActorQuestsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
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

	req := engine.GetActorQuestsRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	if status := OptionalString(params, "status", ""); status != "" {
		s := model.QuestStatus(status)
		req.Status = &s
	}

	result, err := e.GetActorQuests(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("角色有 %d 个任务", len(result.Quests)),
	}, nil
}

// ========== 生活方式与时间工具 ==========

// SetLifestyleTool 设置生活方式
type SetLifestyleTool struct {
	EngineTool
}

func NewSetLifestyleTool(e *engine.Engine) *SetLifestyleTool {
	return &SetLifestyleTool{
		EngineTool: *NewEngineTool(
			"set_lifestyle",
			"设置角色生活方式等级",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"tier": map[string]any{
						"type":        "string",
						"description": "生活方式等级 (wretched=悲惨, squalid=肮脏, poor=贫困, modest=普通, comfortable=舒适, wealthy=富裕, aristocratic=贵族)",
					},
				},
				"required": []string{"game_id", "tier"},
			},
			e,
			false,
		),
	}
}

func (t *SetLifestyleTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	tierStr, err := RequireString(params, "tier")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	tier := model.LifestyleTier(tierStr)

	req := engine.SetLifestyleRequest{
		GameID: gameID,
		Tier:   tier,
	}

	result, err := e.SetLifestyle(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: result.Message,
	}, nil
}

// AdvanceGameTimeTool 推进游戏时间
type AdvanceGameTimeTool struct {
	EngineTool
}

func NewAdvanceGameTimeTool(e *engine.Engine) *AdvanceGameTimeTool {
	return &AdvanceGameTimeTool{
		EngineTool: *NewEngineTool(
			"advance_game_time",
			"推进游戏时间并扣除生活方式开销",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"days": map[string]any{
						"type":        "integer",
						"description": "要推进的天数",
					},
				},
				"required": []string{"game_id", "days"},
			},
			e,
			false,
		),
	}
}

func (t *AdvanceGameTimeTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	gameID := model.ID(gameIDStr)

	days, err := RequireInt(params, "days")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.AdvanceGameTimeRequest{
		GameID: gameID,
		Days:   days,
	}

	result, err := e.AdvanceGameTime(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: result.PaymentSuccess,
		Data:    result,
		Message: result.Message,
	}, nil
}
