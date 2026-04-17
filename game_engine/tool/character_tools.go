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
						"description": "种族 (human, elf, dwarf, halfling, dragonborn, gnome, half-elf, half-orc, tiefling)",
					},
					"class": map[string]any{
						"type":        "string",
						"description": "主职业 (野蛮人, 吟游诗人, 牧师, 德鲁伊, 战士, 武僧, 圣武士, 游侠, 游荡者, 术士, 邪术师, 法师)",
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
						"description": "背景（可选）",
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

	gameID := model.ID(params["game_id"].(string))
	name := params["name"].(string)
	race := params["race"].(string)
	class := params["class"].(string)

	level := 1
	if l, ok := params["level"].(float64); ok {
		level = int(l)
	}

	scores := params["ability_scores"].(map[string]any)
	abilityScores := engine.AbilityScoresInput{
		Strength:     int(scores["strength"].(float64)),
		Dexterity:    int(scores["dexterity"].(float64)),
		Constitution: int(scores["constitution"].(float64)),
		Intelligence: int(scores["intelligence"].(float64)),
		Wisdom:       int(scores["wisdom"].(float64)),
		Charisma:     int(scores["charisma"].(float64)),
	}

	alignment := "True Neutral"
	if a, ok := params["alignment"].(string); ok {
		alignment = a
	}

	pc := &engine.PlayerCharacterInput{
		Name:          name,
		Race:          race,
		Class:         class,
		Level:         level,
		Alignment:     alignment,
		AbilityScores: abilityScores,
	}

	if bg, ok := params["background"].(string); ok {
		pc.Background = bg
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

	gameID := model.ID(params["game_id"].(string))
	name := params["name"].(string)

	npc := &engine.NPCInput{
		Name: name,
	}

	if desc, ok := params["description"].(string); ok {
		npc.Description = desc
	}
	if occ, ok := params["occupation"].(string); ok {
		npc.Occupation = occ
	}
	if att, ok := params["attitude"].(string); ok {
		npc.Attitude = att
	}
	if ct, ok := params["creature_type"].(string); ok {
		npc.CreatureType = ct
	}
	if scores, ok := params["ability_scores"].(map[string]any); ok {
		npc.AbilityScores = engine.AbilityScoresInput{
			Strength:     int(scores["strength"].(float64)),
			Dexterity:    int(scores["dexterity"].(float64)),
			Constitution: int(scores["constitution"].(float64)),
			Intelligence: int(scores["intelligence"].(float64)),
			Wisdom:       int(scores["wisdom"].(float64)),
			Charisma:     int(scores["charisma"].(float64)),
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

	gameID := model.ID(params["game_id"].(string))
	name := params["name"].(string)

	enemy := &engine.EnemyInput{
		Name: name,
	}

	if hp, ok := params["hit_points"].(float64); ok {
		enemy.HitPoints = int(hp)
	}
	if ac, ok := params["armor_class"].(float64); ok {
		enemy.ArmorClass = int(ac)
	}
	if cr, ok := params["challenge_rating"].(string); ok {
		enemy.ChallengeRating = cr
	}
	if ct, ok := params["creature_type"].(string); ok {
		enemy.CreatureType = ct
	}
	if xp, ok := params["xp_value"].(float64); ok {
		enemy.XPValue = int(xp)
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

	gameID := model.ID(params["game_id"].(string))
	name := params["name"].(string)

	companion := &engine.CompanionInput{
		Name: name,
	}

	if lid, ok := params["leader_id"].(string); ok {
		companion.LeaderID = lid
	}
	if loyalty, ok := params["loyalty"].(float64); ok {
		companion.Loyalty = int(loyalty)
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
			"获取角色的基本信息",
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

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))

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
			"获取玩家角色(PC)的详细信息",
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

	gameID := model.ID(params["game_id"].(string))
	pcID := model.ID(params["pc_id"].(string))

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
			"列出游戏中的所有角色",
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

	gameID := model.ID(params["game_id"].(string))

	req := engine.ListActorsRequest{
		GameID: gameID,
	}

	if tf, ok := params["type_filter"].(string); ok && tf != "" {
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

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))
	updates := params["updates"].(map[string]any)

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

	err := e.UpdateActor(ctx, req)
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

	gameID := model.ID(params["game_id"].(string))
	actorID := model.ID(params["actor_id"].(string))

	req := engine.RemoveActorRequest{
		GameID:  gameID,
		ActorID: actorID,
	}

	err := e.RemoveActor(ctx, req)
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

	gameID := model.ID(params["game_id"].(string))
	pcID := model.ID(params["pc_id"].(string))
	xp := int(params["xp"].(float64))

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
