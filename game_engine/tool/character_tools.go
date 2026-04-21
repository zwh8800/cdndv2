package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// CreatePCTool 创建玩家角色
type CreatePCTool struct {
	EngineTool
}

func NewCreatePCTool(e *engine.Engine) *CreatePCTool {
	return &CreatePCTool{
		EngineTool: *NewEngineTool(
			"create_pc",
			"创建一个新的玩家角色(PC)",
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
						"description": "种族名称，必须使用以下中文标准名之一",
						"enum":        []string{"人类", "精灵", "矮人", "半身人", "龙裔", "侏儒", "半精灵", "半兽人", "提夫林"},
					},
					"class": map[string]any{
						"type":        "string",
						"description": "主职业，必须使用以下中文标准名之一",
						"enum":        []string{"野蛮人", "吟游诗人", "牧师", "德鲁伊", "战士", "武僧", "圣武士", "游侠", "游荡者", "术士", "邪术师", "法师"},
					},
					"level": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"description": "角色等级（默认1）",
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
					"background": map[string]any{
						"type":        "string",
						"description": "背景名称，必须使用以下中文标准名之一",
						"enum":        []string{"侍僧", "罪犯", "学者", "士兵"},
					},
					"alignment": map[string]any{
						"type":        "string",
						"description": "阵营（如 Lawful Good, True Neutral 等）",
					},
				},
				"required": []string{"game_id", "name", "race", "class", "ability_scores"},
			},
			e,
			false,
		),
	}
}

func (t *CreatePCTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
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

	level := OptionalInt(params, "level", 1)

	scores, err := RequireMap(params, "ability_scores")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	abilityScores := engine.AbilityScoresInput{
		Strength:     OptionalInt(scores, "strength", 10),
		Dexterity:    OptionalInt(scores, "dexterity", 10),
		Constitution: OptionalInt(scores, "constitution", 10),
		Intelligence: OptionalInt(scores, "intelligence", 10),
		Wisdom:       OptionalInt(scores, "wisdom", 10),
		Charisma:     OptionalInt(scores, "charisma", 10),
	}

	alignment := OptionalString(params, "alignment", "True Neutral")
	background := OptionalString(params, "background", "")

	gameID := model.ID(gameIDStr)

	pc := &engine.PlayerCharacterInput{
		Name:          name,
		Race:          race,
		Class:         class,
		Level:         level,
		Alignment:     alignment,
		AbilityScores: abilityScores,
		Background:    background,
	}

	req := engine.CreatePCRequest{
		GameID: gameID,
		PC:     pc,
	}

	result, err := e.CreatePC(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data: map[string]any{
			"actor_id":    string(result.Actor.ID),
			"name":        result.Actor.Name,
			"hit_points":  result.Actor.HitPoints,
			"armor_class": result.Actor.ArmorClass,
			"speed":       result.Actor.Speed,
		},
		Message: "成功创建角色 " + result.Actor.Name,
	}, nil
}

// CreateNPCTool 创建NPC
type CreateNPCTool struct {
	EngineTool
}

func NewCreateNPCTool(e *engine.Engine) *CreateNPCTool {
	return &CreateNPCTool{
		EngineTool: *NewEngineTool(
			"create_npc",
			"创建一个非玩家角色(NPC)",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "NPC名称",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "NPC描述",
					},
					"occupation": map[string]any{
						"type":        "string",
						"description": "职业/身份",
					},
					"attitude": map[string]any{
						"type":        "string",
						"description": "对玩家的态度",
					},
					"creature_type": map[string]any{
						"type":        "string",
						"description": "生物类型",
					},
					"ability_scores": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"strength":     map[string]any{"type": "integer"},
							"dexterity":    map[string]any{"type": "integer"},
							"constitution": map[string]any{"type": "integer"},
							"intelligence": map[string]any{"type": "integer"},
							"wisdom":       map[string]any{"type": "integer"},
							"charisma":     map[string]any{"type": "integer"},
						},
					},
				},
				"required": []string{"game_id", "name"},
			},
			e,
			false,
		),
	}
}

func (t *CreateNPCTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	name, err := RequireString(params, "name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)

	npc := &engine.NPCInput{
		Name:         name,
		Description:  OptionalString(params, "description", ""),
		Occupation:   OptionalString(params, "occupation", ""),
		Attitude:     OptionalString(params, "attitude", ""),
		CreatureType: OptionalString(params, "creature_type", ""),
	}

	if scores, ok := params["ability_scores"].(map[string]any); ok {
		npc.AbilityScores = engine.AbilityScoresInput{
			Strength:     OptionalInt(scores, "strength", 10),
			Dexterity:    OptionalInt(scores, "dexterity", 10),
			Constitution: OptionalInt(scores, "constitution", 10),
			Intelligence: OptionalInt(scores, "intelligence", 10),
			Wisdom:       OptionalInt(scores, "wisdom", 10),
			Charisma:     OptionalInt(scores, "charisma", 10),
		}
	}

	req := engine.CreateNPCRequest{
		GameID: gameID,
		NPC:    npc,
	}

	result, err := e.CreateNPC(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data: map[string]any{
			"actor_id": string(result.Actor.ID),
			"name":     result.Actor.Name,
		},
		Message: "成功创建NPC " + result.Actor.Name,
	}, nil
}

// CreateEnemyTool 创建敌人
type CreateEnemyTool struct {
	EngineTool
}

func NewCreateEnemyTool(e *engine.Engine) *CreateEnemyTool {
	return &CreateEnemyTool{
		EngineTool: *NewEngineTool(
			"create_enemy",
			"创建一个敌人/怪物",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "敌人名称",
					},
					"hit_points": map[string]any{
						"type":        "integer",
						"description": "生命值",
					},
					"armor_class": map[string]any{
						"type":        "integer",
						"description": "护甲等级",
					},
					"challenge_rating": map[string]any{
						"type":        "string",
						"description": "挑战等级（如 1/4, 1, 5）",
					},
					"creature_type": map[string]any{
						"type":        "string",
						"description": "生物类型",
					},
					"xp_value": map[string]any{
						"type":        "integer",
						"description": "经验值",
					},
				},
				"required": []string{"game_id", "name"},
			},
			e,
			false,
		),
	}
}

func (t *CreateEnemyTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	name, err := RequireString(params, "name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)

	enemy := &engine.EnemyInput{
		Name:            name,
		HitPoints:       OptionalInt(params, "hit_points", 0),
		ArmorClass:      OptionalInt(params, "armor_class", 0),
		ChallengeRating: OptionalString(params, "challenge_rating", ""),
		CreatureType:    OptionalString(params, "creature_type", ""),
		XPValue:         OptionalInt(params, "xp_value", 0),
	}

	req := engine.CreateEnemyRequest{
		GameID: gameID,
		Enemy:  enemy,
	}

	result, err := e.CreateEnemy(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data: map[string]any{
			"actor_id": string(result.Actor.ID),
			"name":     result.Actor.Name,
		},
		Message: "成功创建敌人 " + result.Actor.Name,
	}, nil
}

// CreateCompanionTool 创建同伴
type CreateCompanionTool struct {
	EngineTool
}

func NewCreateCompanionTool(e *engine.Engine) *CreateCompanionTool {
	return &CreateCompanionTool{
		EngineTool: *NewEngineTool(
			"create_companion",
			"创建一个同伴/盟友",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "同伴名称",
					},
					"leader_id": map[string]any{
						"type":        "string",
						"description": "领导者（玩家）ID",
					},
					"loyalty": map[string]any{
						"type":        "integer",
						"description": "忠诚度",
					},
				},
				"required": []string{"game_id", "name"},
			},
			e,
			false,
		),
	}
}

func (t *CreateCompanionTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	name, err := RequireString(params, "name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)

	companion := &engine.CompanionInput{
		Name:     name,
		LeaderID: OptionalString(params, "leader_id", ""),
		Loyalty:  OptionalInt(params, "loyalty", 0),
	}

	req := engine.CreateCompanionRequest{
		GameID:    gameID,
		Companion: companion,
	}

	result, err := e.CreateCompanion(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data: map[string]any{
			"actor_id": string(result.Actor.ID),
			"name":     result.Actor.Name,
		},
		Message: "成功创建同伴 " + result.Actor.Name,
	}, nil
}

// GetActorTool 获取角色信息
type GetActorTool struct {
	EngineTool
}

func NewGetActorTool(e *engine.Engine) *GetActorTool {
	return &GetActorTool{
		EngineTool: *NewEngineTool(
			"get_actor",
			"获取任意角色（PC/NPC/Enemy/Companion）的基本信息，包括名称、类型、HP、AC、速度、属性值等。Use when: 需要查看某个角色的当前状态（生命值、护甲等级等）；需要确认角色是否存在或获取其 actor_id。Do NOT use when: 需要 PC 的详细信息如法术列表、装备等（用 get_pc）；需要列出所有角色（用 list_actors）；需要获取 NPC 的态度（用 get_npc_attitude）。",
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
				},
				"required": []string{"game_id", "actor_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetActorTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)

	req := engine.GetActorRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	result, err := e.GetActor(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.Actor,
		Message: "获取角色 " + result.Actor.Name + " 的信息",
	}, nil
}

// GetPCTool 获取玩家角色详情
type GetPCTool struct {
	EngineTool
}

func NewGetPCTool(e *engine.Engine) *GetPCTool {
	return &GetPCTool{
		EngineTool: *NewEngineTool(
			"get_pc",
			"获取玩家角色(PC)的完整详细信息，包括种族、职业、等级、属性值、技能、法术位、特性等。Use when: 需要 PC 的完整角色卡信息；需要确认 PC 的法术列表、技能修正等详细数据。Do NOT use when: 只需要角色基本信息如 HP/AC（用 get_actor）；需要列出所有角色（用 list_actors）。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"pc_id": map[string]any{
						"type":        "string",
						"description": "玩家角色ID",
					},
				},
				"required": []string{"game_id", "pc_id"},
			},
			e,
			true,
		),
	}
}

func (t *GetPCTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	pcIDStr, err := RequireString(params, "pc_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	pcID := model.ID(pcIDStr)

	req := engine.GetPCRequest{
		GameID: gameID,
		PCID:   pcID,
	}

	result, err := e.GetPC(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.PC,
		Message: "获取玩家角色 " + result.PC.Name + " 的详细信息",
	}, nil
}

// ListActorsTool 列出所有角色
type ListActorsTool struct {
	EngineTool
}

func NewListActorsTool(e *engine.Engine) *ListActorsTool {
	return &ListActorsTool{
		EngineTool: *NewEngineTool(
			"list_actors",
			"列出游戏中的所有角色（PC/NPC/Enemy/Companion），可通过 type_filter 按类型筛选。Use when: 需要查看场景中有哪些角色；需要获取角色ID列表以进行后续操作（如攻击目标选择）。Do NOT use when: 只需要某个特定角色的信息（用 get_actor）；需要 PC 的详细信息（用 get_pc）。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"type_filter": map[string]any{
						"type":        "string",
						"description": "角色类型过滤 (pc, npc, enemy, companion)",
					},
				},
				"required": []string{"game_id"},
			},
			e,
			true,
		),
	}
}

func (t *ListActorsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)

	req := engine.ListActorsRequest{
		GameID: gameID,
	}

	if tf := OptionalString(params, "type_filter", ""); tf != "" {
		req.Filter = &engine.ActorFilter{
			Types: []model.ActorType{model.ActorType(tf)},
		}
	}

	result, err := e.ListActors(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result.Actors,
		Message: fmt.Sprintf("找到 %d 个角色", len(result.Actors)),
	}, nil
}

// UpdateActorTool 更新角色状态
type UpdateActorTool struct {
	EngineTool
}

func NewUpdateActorTool(e *engine.Engine) *UpdateActorTool {
	return &UpdateActorTool{
		EngineTool: *NewEngineTool(
			"update_actor",
			"更新角色的状态",
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
					"updates": map[string]any{
						"type":        "object",
						"description": "更新内容",
					},
				},
				"required": []string{"game_id", "actor_id", "updates"},
			},
			e,
			false,
		),
	}
}

func (t *UpdateActorTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	updates, err := RequireMap(params, "updates")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)

	update := engine.ActorUpdate{}

	if hp, ok := updates["hit_points"].(map[string]any); ok {
		hpUpdate := engine.HitPointUpdate{}
		if curr, ok := hp["current"].(float64); ok {
			v := int(curr)
			hpUpdate.Current = &v
		}
		if tmp, ok := hp["temp_hit_points"].(float64); ok {
			v := int(tmp)
			hpUpdate.TempHitPoints = &v
		}
		update.HitPoints = &hpUpdate
	}

	if pos, ok := updates["position"].(map[string]any); ok {
		p := &model.Point{}
		if x, ok := pos["x"].(float64); ok {
			p.X = int(x)
		}
		if y, ok := pos["y"].(float64); ok {
			p.Y = int(y)
		}
		update.Position = p
	}

	req := engine.UpdateActorRequest{
		GameID:  gameID,
		ActorID: actorID,
		Update:  update,
	}

	err = e.UpdateActor(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: "角色状态已更新",
	}, nil
}

// RemoveActorTool 移除角色
type RemoveActorTool struct {
	EngineTool
}

func NewRemoveActorTool(e *engine.Engine) *RemoveActorTool {
	return &RemoveActorTool{
		EngineTool: *NewEngineTool(
			"remove_actor",
			"从游戏中移除一个角色",
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
				},
				"required": []string{"game_id", "actor_id"},
			},
			e,
			false,
		),
	}
}

func (t *RemoveActorTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	actorIDStr, err := RequireString(params, "actor_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	actorID := model.ID(actorIDStr)

	req := engine.RemoveActorRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	err = e.RemoveActor(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Message: "角色已从游戏中移除",
	}, nil
}

// AddExperienceTool 添加经验值
type AddExperienceTool struct {
	EngineTool
}

func NewAddExperienceTool(e *engine.Engine) *AddExperienceTool {
	return &AddExperienceTool{
		EngineTool: *NewEngineTool(
			"add_experience",
			"为玩家角色添加经验值",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"game_id": map[string]any{
						"type":        "string",
						"description": "游戏会话ID",
					},
					"pc_id": map[string]any{
						"type":        "string",
						"description": "玩家角色ID",
					},
					"xp": map[string]any{
						"type":        "integer",
						"description": "添加的经验值",
					},
				},
				"required": []string{"game_id", "pc_id", "xp"},
			},
			e,
			false,
		),
	}
}

func (t *AddExperienceTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	gameIDStr, err := RequireString(params, "game_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	pcIDStr, err := RequireString(params, "pc_id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	xp, err := RequireInt(params, "xp")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	gameID := model.ID(gameIDStr)
	pcID := model.ID(pcIDStr)

	req := engine.AddExperienceRequest{
		GameID: gameID,
		PCID:   pcID,
		XP:     xp,
	}

	result, err := e.AddExperience(ctx, req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	msg := fmt.Sprintf("添加了 %d 点经验值", xp)
	if result.LeveledUp {
		msg += fmt.Sprintf("，角色升级到 %d 级！", result.NewLevel)
	}

	return &ToolResult{
		Success: true,
		Data: map[string]any{
			"leveled_up": result.LeveledUp,
			"old_level":  result.OldLevel,
			"new_level":  result.NewLevel,
		},
		Message: msg,
	}, nil
}

// ========== 复合工具 - 角色创建 ==========

// NewQueryRacesTool 复合查询种族工具：返回种族列表含简要描述
//
// 合并: list_races + get_race
func NewQueryRacesTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name_filter": map[string]any{
				"type":        "string",
				"description": "按名称过滤（可选，留空返回所有）",
			},
		},
	}

	desc := `Query available playable races with brief descriptions.

Use when: Player is creating a character and wants to see available race options. One call returns all races with their key traits.

Do NOT use when: Player already knows which race they want and just needs detailed data for character creation.

Parameters:
  - name_filter: Optional filter by name (contains)

Returns: List of races with name, description, and key traits.`

	steps := []ToolStep{
		{
			ToolName: "list_races",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return map[string]any{}
			},
		},
	}

	return NewCompositeTool(
		"query_races",
		desc,
		schema,
		registry,
		steps,
		true, // read only
	)
}

// NewQueryClassesTool 复合查询职业工具：返回职业列表含简要描述
//
// 合并: list_classes + get_class
func NewQueryClassesTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name_filter": map[string]any{
				"type":        "string",
				"description": "按名称过滤（可选，留空返回所有）",
			},
		},
	}

	desc := `Query available character classes with brief descriptions.

Use when: Player is creating a character and wants to see available class options. One call returns all classes with their primary abilities and hit dice.

Do NOT use when: Player already knows which class they want.

Parameters:
  - name_filter: Optional filter by name (contains)

Returns: List of classes with name, description, and key features.`

	steps := []ToolStep{
		{
			ToolName: "list_classes",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return map[string]any{}
			},
		},
	}

	return NewCompositeTool(
		"query_classes",
		desc,
		schema,
		registry,
		steps,
		true,
	)
}

// NewQueryBackgroundsTool 复合查询背景工具：返回背景列表含简要描述
//
// 合并: list_backgrounds + get_background
func NewQueryBackgroundsTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name_filter": map[string]any{
				"type":        "string",
				"description": "按名称过滤（可选，留空返回所有）",
			},
		},
	}

	desc := `Query available character backgrounds with brief descriptions.

Use when: Player is creating a character and wants to see available background options. One call returns all backgrounds with their feature descriptions.

Do NOT use when: Player already knows which background they want.

Parameters:
  - name_filter: Optional filter by name (contains)

Returns: List of backgrounds with name, description, and feature.`

	steps := []ToolStep{
		{
			ToolName: "list_backgrounds",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return map[string]any{}
			},
		},
	}

	return NewCompositeTool(
		"query_backgrounds",
		desc,
		schema,
		registry,
		steps,
		true,
	)
}

// NewCreateCharacterTool 增强版创建角色复合工具：一步创建角色并自动分配初始装备
//
// 增强: create_pc + 自动初始装备
func NewCreateCharacterTool(e *engine.Engine, registry *ToolRegistry) *CompositeTool {
	schema := map[string]any{
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
				"description": "种族名称",
			},
			"class": map[string]any{
				"type":        "string",
				"description": "职业名称",
			},
			"level": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "角色等级，默认 1",
			},
			"background": map[string]any{
				"type":        "string",
				"description": "背景名称",
			},
			"alignment": map[string]any{
				"type":        "string",
				"description": "阵营",
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
		},
		"required": []string{"game_id", "name", "race", "class", "ability_scores"},
	}

	desc := `Create a new player character in one step. Automatically handles creation and returns complete character info.

Use when: Player wants to create a new character. One call completes the entire process.

Do NOT use when: Just browsing available options (use query_races/query_classes first).

Parameters:
  - game_id: Game session ID
  - name: Character name
  - race: Race name
  - class: Class name
  - level: Starting level (default 1)
  - background: Background name
  - alignment: Alignment
  - ability_scores: Six ability scores (str/dex/con/int/wis/cha)

Returns: Complete character info with ID, HP, AC, speed.`

	steps := []ToolStep{
		{
			ToolName: "create_pc",
			Params: func(ctx context.Context, params map[string]any, prevResults map[string]*ToolResult) map[string]any {
				return params
			},
		},
	}

	return NewCompositeTool(
		"create_character",
		desc,
		schema,
		registry,
		steps,
		false,
	)
}
