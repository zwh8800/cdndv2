# Tool定义设计

## 0. 核心原则

> **game_engine 绝不自行运算任何游戏逻辑。**
> Tool 是 game_engine 与 dnd 引擎之间的桥梁，每个 Tool 的实现必须直接调用引擎API，
> 不得在 Tool 层自行计算任何规则结果。Tool 仅负责参数组装、API调用、结果格式化。

## 1. Tool接口定义

### 1.1 基础接口

```go
// Tool 基础接口
type Tool interface {
    // Name 返回Tool名称
    Name() string

    // Description 返回Tool描述
    Description() string

    // ParametersSchema 返回参数JSON Schema
    ParametersSchema() map[string]any

    // Execute 执行Tool
    Execute(ctx context.Context, params map[string]any) (*ToolResult, error)
}

// ToolResult Tool执行结果
type ToolResult struct {
    Success   bool            `json:"success"`
    Data      any             `json:"data"`
    Message   string          `json:"message"`
    Error     string          `json:"error,omitempty"`
    Metadata  map[string]any  `json:"metadata,omitempty"`
}
```

### 1.2 基础Tool实现

```go
// BaseTool 基础Tool实现
type BaseTool struct {
    name        string
    description string
    schema      map[string]any
}

func (t *BaseTool) Name() string {
    return t.name
}

func (t *BaseTool) Description() string {
    return t.description
}

func (t *BaseTool) ParametersSchema() map[string]any {
    return t.schema
}
```

### 1.3 引擎Tool基类

```go
// EngineTool 引擎Tool基类
type EngineTool struct {
    BaseTool
    engine *engine.Engine
}

func NewEngineTool(name, description string, schema map[string]any, e *engine.Engine) *EngineTool {
    return &EngineTool{
        BaseTool: BaseTool{
            name:        name,
            description: description,
            schema:      schema,
        },
        engine: e,
    }
}
```

## 2. Tool分类

### 2.1 按功能分类

| 分类 | Tool数量 | 说明 |
|------|----------|------|
| 游戏会话 | 6 | 创建、加载、保存、删除、列出、获取游戏 |
| 角色管理 | 10 | 创建、查询、更新、移除角色 |
| 角色升级 | 2 | 经验值、等级提升 |
| 休息系统 | 3 | 短休、长休 |
| 战斗系统 | 12 | 战斗流程、动作执行、伤害治疗 |
| 法术系统 | 11 | 施法、法术位、专注管理、仪式施法 |
| 检定系统 | 5 | 属性、技能、豁免检定 |
| 库存管理 | 9 | 物品、装备、货币管理 |
| 专长系统 | 5 | 专长查询、选择、移除 |
| 场景管理 | 14 | 场景CRUD、连接、移动 |
| 探索系统 | 4 | 旅行、觅食、导航 |
| 社交互动 | 2 | NPC互动、态度查询 |
| 任务系统 | 10 | 任务CRUD、进度更新 |
| 死亡豁免 | 3 | 死亡豁免、稳定、状态查询 |
| 背景系统 | 2 | 应用背景、获取特性 |
| 制作系统 | 4 | 开始、推进、完成制作 |
| 诅咒系统 | 3 | 施加、移除、查询诅咒 |
| 环境系统 | 2 | 设置环境、结算环境伤害 |
| 力竭系统 | 3 | 施加、移除、查询力竭 |
| 骑乘系统 | 3 | 骑乘、解除、速度计算 |
| 移动系统 | 5 | 跳跃、跌落、闭气、窒息、遭遇检定 |
| 毒药系统 | 3 | 施加、结算、移除毒药 |
| 陷阱系统 | 4 | 放置、侦测、解除、触发陷阱 |
| 魔法物品 | 4 | 使用、调谐、解除调谐、充能 |
| 多职业系统 | 2 | 验证多职业、获取法术位 |
| 生活方式 | 2 | 设置生活方式、推进时间 |
| 骰子系统 | 5 | 各种骰子投掷 |
| 数据查询 | 36 | 种族、职业、法术、装备等数据查询 |
| 怪物系统 | 1 | 加载怪物模板 |
| 信息聚合 | 4 | 生活方式、制作、负重、力竭效果查询 |
| 状态查询 | 3 | 状态摘要、角色卡、战斗摘要 |
| 阶段管理 | 3 | 设置、获取阶段、获取允许操作 |

### 2.2 按操作类型分类

| 类型 | 说明 | 示例 |
|------|------|------|
| 查询类 | 获取信息，不改变状态 | get_actor, list_scenes |
| 操作类 | 改变游戏状态 | create_pc, execute_attack |
| 流程类 | 控制游戏流程 | start_combat, next_turn |

## 3. Tool Schema定义

### 3.1 游戏会话Tools

#### new_game

```json
{
    "name": "new_game",
    "description": "创建一个新的游戏会话",
    "parameters": {
        "type": "object",
        "properties": {
            "name": {
                "type": "string",
                "description": "游戏名称"
            },
            "setting": {
                "type": "string",
                "description": "游戏背景设定（可选）"
            }
        },
        "required": ["name"]
    }
}
```

#### load_game

```json
{
    "name": "load_game",
    "description": "加载已存在的游戏存档",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏ID"
            }
        },
        "required": ["game_id"]
    }
}
```

#### save_game

```json
{
    "name": "save_game",
    "description": "保存当前游戏状态",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏ID"
            }
        },
        "required": ["game_id"]
    }
}
```

### 3.2 角色管理Tools

#### create_pc

```json
{
    "name": "create_pc",
    "description": "创建一个新的玩家角色(PC)",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "name": {
                "type": "string",
                "description": "角色名称"
            },
            "race": {
                "type": "string",
                "enum": ["human", "elf", "dwarf", "halfling", "dragonborn", "gnome", "half-elf", "half-orc", "tiefling"],
                "description": "种族"
            },
            "classes": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "class": {
                            "type": "string",
                            "description": "职业名称"
                        },
                        "level": {
                            "type": "integer",
                            "minimum": 1,
                            "description": "职业等级"
                        }
                    }
                },
                "description": "职业列表"
            },
            "ability_scores": {
                "type": "object",
                "properties": {
                    "strength": {"type": "integer", "minimum": 1, "maximum": 20},
                    "dexterity": {"type": "integer", "minimum": 1, "maximum": 20},
                    "constitution": {"type": "integer", "minimum": 1, "maximum": 20},
                    "intelligence": {"type": "integer", "minimum": 1, "maximum": 20},
                    "wisdom": {"type": "integer", "minimum": 1, "maximum": 20},
                    "charisma": {"type": "integer", "minimum": 1, "maximum": 20}
                },
                "required": ["strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma"],
                "description": "六项属性值"
            },
            "background": {
                "type": "string",
                "description": "背景（可选）"
            }
        },
        "required": ["game_id", "name", "race", "classes", "ability_scores"]
    }
}
```

#### get_actor

```json
{
    "name": "get_actor",
    "description": "获取角色的基本信息",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "actor_id": {
                "type": "string",
                "description": "角色ID"
            }
        },
        "required": ["game_id", "actor_id"]
    }
}
```

#### update_actor

```json
{
    "name": "update_actor",
    "description": "更新角色的状态",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "actor_id": {
                "type": "string",
                "description": "角色ID"
            },
            "updates": {
                "type": "object",
                "properties": {
                    "hit_points": {
                        "type": "object",
                        "properties": {
                            "current": {"type": "integer"},
                            "temporary": {"type": "integer"}
                        },
                        "description": "生命值更新"
                    },
                    "conditions": {
                        "type": "object",
                        "properties": {
                            "add": {
                                "type": "array",
                                "items": {"type": "string"},
                                "description": "添加的状态"
                            },
                            "remove": {
                                "type": "array",
                                "items": {"type": "string"},
                                "description": "移除的状态"
                            }
                        },
                        "description": "状态效果更新"
                    },
                    "position": {
                        "type": "object",
                        "properties": {
                            "x": {"type": "integer"},
                            "y": {"type": "integer"}
                        },
                        "description": "位置更新"
                    }
                },
                "description": "更新内容"
            }
        },
        "required": ["game_id", "actor_id", "updates"]
    }
}
```

### 3.3 战斗系统Tools

#### start_combat

```json
{
    "name": "start_combat",
    "description": "开始一场战斗遭遇",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "scene_id": {
                "type": "string",
                "description": "战斗发生的场景ID"
            },
            "combatants": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "actor_id": {"type": "string", "description": "参战者ID"},
                        "team": {"type": "string", "description": "队伍标识"}
                    }
                },
                "description": "参战者列表"
            }
        },
        "required": ["game_id", "scene_id", "combatants"]
    }
}
```

#### execute_attack

```json
{
    "name": "execute_attack",
    "description": "执行一次攻击动作",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "attacker_id": {
                "type": "string",
                "description": "攻击者ID"
            },
            "target_id": {
                "type": "string",
                "description": "目标ID"
            },
            "attack": {
                "type": "object",
                "properties": {
                    "weapon_id": {
                        "type": "string",
                        "description": "武器ID（可选，徒手攻击不需要）"
                    },
                    "is_unarmed": {
                        "type": "boolean",
                        "description": "是否徒手攻击"
                    },
                    "is_off_hand": {
                        "type": "boolean",
                        "description": "是否为副手攻击"
                    },
                    "advantage": {
                        "type": "string",
                        "enum": ["none", "advantage", "disadvantage"],
                        "description": "优势/劣势"
                    }
                },
                "description": "攻击参数"
            }
        },
        "required": ["game_id", "attacker_id", "target_id", "attack"]
    }
}
```

#### next_turn

```json
{
    "name": "next_turn",
    "description": "推进到下一个角色的回合",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            }
        },
        "required": ["game_id"]
    }
}
```

### 3.4 检定系统Tools

#### perform_ability_check

```json
{
    "name": "perform_ability_check",
    "description": "执行一次属性检定",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "actor_id": {
                "type": "string",
                "description": "进行检定的角色ID"
            },
            "ability": {
                "type": "string",
                "enum": ["strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma"],
                "description": "检定的属性"
            },
            "dc": {
                "type": "integer",
                "minimum": 1,
                "description": "难度等级"
            },
            "advantage": {
                "type": "string",
                "enum": ["none", "advantage", "disadvantage"],
                "description": "优势/劣势"
            },
            "reason": {
                "type": "string",
                "description": "检定原因（可选）"
            }
        },
        "required": ["game_id", "actor_id", "ability", "dc"]
    }
}
```

#### perform_skill_check

```json
{
    "name": "perform_skill_check",
    "description": "执行一次技能检定",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "actor_id": {
                "type": "string",
                "description": "进行检定的角色ID"
            },
            "skill": {
                "type": "string",
                "enum": ["acrobatics", "animal_handling", "arcana", "athletics", "deception", "history", "insight", "intimidation", "investigation", "medicine", "nature", "perception", "performance", "persuasion", "religion", "sleight_of_hand", "stealth", "survival"],
                "description": "技能名称"
            },
            "dc": {
                "type": "integer",
                "minimum": 1,
                "description": "难度等级"
            },
            "advantage": {
                "type": "string",
                "enum": ["none", "advantage", "disadvantage"],
                "description": "优势/劣势"
            }
        },
        "required": ["game_id", "actor_id", "skill", "dc"]
    }
}
```

### 3.5 法术系统Tools

#### cast_spell

```json
{
    "name": "cast_spell",
    "description": "施放一个法术",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "caster_id": {
                "type": "string",
                "description": "施法者ID"
            },
            "spell": {
                "type": "object",
                "properties": {
                    "spell_id": {
                        "type": "string",
                        "description": "法术ID"
                    },
                    "level": {
                        "type": "integer",
                        "description": "施法等级（可选，用于升阶施法）"
                    },
                    "targets": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "目标ID列表"
                    },
                    "point": {
                        "type": "object",
                        "properties": {
                            "x": {"type": "integer"},
                            "y": {"type": "integer"}
                        },
                        "description": "目标点坐标"
                    }
                },
                "required": ["spell_id"],
                "description": "法术参数"
            }
        },
        "required": ["game_id", "caster_id", "spell"]
    }
}
```

#### get_spell_slots

```json
{
    "name": "get_spell_slots",
    "description": "获取施法者的法术位状态",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "caster_id": {
                "type": "string",
                "description": "施法者ID"
            }
        },
        "required": ["game_id", "caster_id"]
    }
}
```

### 3.6 场景管理Tools

#### create_scene

```json
{
    "name": "create_scene",
    "description": "创建一个新场景",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "name": {
                "type": "string",
                "description": "场景名称"
            },
            "description": {
                "type": "string",
                "description": "场景描述"
            },
            "size": {
                "type": "object",
                "properties": {
                    "width": {"type": "integer"},
                    "height": {"type": "integer"}
                },
                "description": "场景尺寸"
            },
            "lighting": {
                "type": "string",
                "enum": ["bright", "dim", "darkness"],
                "description": "光照条件"
            },
            "terrain_type": {
                "type": "string",
                "description": "地形类型"
            }
        },
        "required": ["game_id", "name"]
    }
}
```

#### get_current_scene

```json
{
    "name": "get_current_scene",
    "description": "获取当前场景信息",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            }
        },
        "required": ["game_id"]
    }
}
```

### 3.7 任务系统Tools

#### create_quest

```json
{
    "name": "create_quest",
    "description": "创建一个新任务",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "title": {
                "type": "string",
                "description": "任务标题"
            },
            "description": {
                "type": "string",
                "description": "任务描述"
            },
            "objectives": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "description": {"type": "string"},
                        "target": {"type": "integer"},
                        "current": {"type": "integer"}
                    }
                },
                "description": "任务目标列表"
            },
            "rewards": {
                "type": "object",
                "properties": {
                    "experience": {"type": "integer"},
                    "gold": {"type": "integer"},
                    "items": {
                        "type": "array",
                        "items": {"type": "string"}
                    }
                },
                "description": "任务奖励"
            },
            "quest_giver_id": {
                "type": "string",
                "description": "任务发布者ID（可选）"
            }
        },
        "required": ["game_id", "title", "description"]
    }
}
```

#### complete_quest

```json
{
    "name": "complete_quest",
    "description": "完成任务并发放奖励",
    "parameters": {
        "type": "object",
        "properties": {
            "game_id": {
                "type": "string",
                "description": "游戏会话ID"
            },
            "quest_id": {
                "type": "string",
                "description": "任务ID"
            }
        },
        "required": ["game_id", "quest_id"]
    }
}
```

## 4. Tool实现示例

### 4.1 游戏会话Tool

```go
// NewGameTool 创建新游戏Tool
type NewGameTool struct {
    EngineTool
}

func NewNewGameTool(e *engine.Engine) *NewGameTool {
    return &NewGameTool{
        EngineTool: *NewEngineTool(
            "new_game",
            "创建一个新的游戏会话",
            map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "name": map[string]any{
                        "type":        "string",
                        "description": "游戏名称",
                    },
                    "setting": map[string]any{
                        "type":        "string",
                        "description": "游戏背景设定（可选）",
                    },
                },
                "required": []string{"name"},
            },
            e,
        ),
    }
}

func (t *NewGameTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
    name, _ := params["name"].(string)
    setting, _ := params["setting"].(string)

    req := engine.NewGameRequest{
        Name:    name,
        Setting: setting,
    }

    result, err := t.engine.NewGame(ctx, req)
    if err != nil {
        return &ToolResult{
            Success: false,
            Error:   err.Error(),
        }, nil
    }

    return &ToolResult{
        Success: true,
        Data: map[string]any{
            "game_id":   result.GameID,
            "game_name": result.GameName,
        },
        Message: result.Message,
    }, nil
}
```

### 4.2 角色管理Tool

```go
// CreatePCTool 创建玩家角色Tool
type CreatePCTool struct {
    EngineTool
}

func NewCreatePCTool(e *engine.Engine) *CreatePCTool {
    return &CreatePCTool{
        EngineTool: *NewEngineTool(
            "create_pc",
            "创建一个新的玩家角色(PC)",
            // ... schema
            e,
        ),
    }
}

func (t *CreatePCTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
    gameID := model.ID(params["game_id"].(string))

    // 解析属性值
    scores := params["ability_scores"].(map[string]any)
    abilityScores := engine.AbilityScoresInput{
        Strength:     int(scores["strength"].(float64)),
        Dexterity:    int(scores["dexterity"].(float64)),
        Constitution: int(scores["constitution"].(float64)),
        Intelligence: int(scores["intelligence"].(float64)),
        Wisdom:       int(scores["wisdom"].(float64)),
        Charisma:     int(scores["charisma"].(float64)),
    }

    // 解析职业
    classes := params["classes"].([]any)
    classInputs := make([]engine.ClassLevelInput, len(classes))
    for i, c := range classes {
        classMap := c.(map[string]any)
        classInputs[i] = engine.ClassLevelInput{
            Class: classMap["class"].(string),
            Level: int(classMap["level"].(float64)),
        }
    }

    req := engine.CreatePCRequest{
        GameID:        gameID,
        Name:          params["name"].(string),
        Race:          params["race"].(string),
        Classes:       classInputs,
        AbilityScores: abilityScores,
    }

    if bg, ok := params["background"].(string); ok {
        req.Background = model.BackgroundID(bg)
    }

    result, err := t.engine.CreatePC(ctx, req)
    if err != nil {
        return &ToolResult{
            Success: false,
            Error:   err.Error(),
        }, nil
    }

    return &ToolResult{
        Success: true,
        Data: map[string]any{
            "actor_id":     result.Actor.ID,
            "name":         result.Actor.Name,
            "hit_points":   result.Actor.HitPoints,
            "armor_class":  result.Actor.ArmorClass,
            "speed":        result.Actor.Speed,
        },
        Message: "成功创建角色 " + result.Actor.Name,
    }, nil
}
```

### 4.3 战斗Tool

```go
// ExecuteAttackTool 执行攻击Tool
type ExecuteAttackTool struct {
    EngineTool
}

func (t *ExecuteAttackTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
    gameID := model.ID(params["game_id"].(string))
    attackerID := model.ID(params["attacker_id"].(string))
    targetID := model.ID(params["target_id"].(string))

    attackParams := params["attack"].(map[string]any)

    attack := engine.AttackInput{
        IsUnarmed: attackParams["is_unarmed"].(bool),
    }

    if weaponID, ok := attackParams["weapon_id"].(string); ok {
        id := model.ID(weaponID)
        attack.WeaponID = &id
    }

    if advantage, ok := attackParams["advantage"].(string); ok {
        switch advantage {
        case "advantage":
            attack.Advantage = model.RollAdvantage
        case "disadvantage":
            attack.Advantage = model.RollDisadvantage
        }
    }

    req := engine.ExecuteAttackRequest{
        GameID:     gameID,
        AttackerID: attackerID,
        TargetID:   targetID,
        Attack:     attack,
    }

    result, err := t.engine.ExecuteAttack(ctx, req)
    if err != nil {
        return &ToolResult{
            Success: false,
            Error:   err.Error(),
        }, nil
    }

    return &ToolResult{
        Success: true,
        Data: map[string]any{
            "attack_roll":   result.AttackTotal,
            "target_ac":     result.TargetAC,
            "hit":           result.Hit,
            "is_critical":   result.IsCritical,
            "damage":        result.Damage,
            "message":       result.Message,
        },
        Message: result.Message,
    }, nil
}
```

## 5. Tool注册中心

### 5.1 Registry设计

```go
// ToolRegistry Tool注册中心
type ToolRegistry struct {
    tools    map[string]Tool
    byAgent  map[string][]string // agent -> tool names
    category map[string][]string // category -> tool names
}

func NewToolRegistry() *ToolRegistry {
    return &ToolRegistry{
        tools:    make(map[string]Tool),
        byAgent:  make(map[string][]string),
        category: make(map[string][]string),
    }
}

// Register 注册Tool
func (r *ToolRegistry) Register(tool Tool, agents []string, category string) {
    r.tools[tool.Name()] = tool

    for _, agent := range agents {
        r.byAgent[agent] = append(r.byAgent[agent], tool.Name())
    }

    if category != "" {
        r.category[category] = append(r.category[category], tool.Name())
    }
}

// Get 获取Tool
func (r *ToolRegistry) Get(name string) (Tool, bool) {
    tool, ok := r.tools[name]
    return tool, ok
}

// GetByAgent 获取Agent可用的Tools
func (r *ToolRegistry) GetByAgent(agent string) []Tool {
    names := r.byAgent[agent]
    tools := make([]Tool, len(names))
    for i, name := range names {
        tools[i] = r.tools[name]
    }
    return tools
}

// GetAll 获取所有Tools的Schema
func (r *ToolRegistry) GetAll() []map[string]any {
    var schemas []map[string]any
    for _, tool := range r.tools {
        schemas = append(schemas, map[string]any{
            "type":       "function",
            "function": map[string]any{
                "name":        tool.Name(),
                "description": tool.Description(),
                "parameters":  tool.ParametersSchema(),
            },
        })
    }
    return schemas
}
```

### 5.2 初始化注册

```go
// InitRegistry 初始化Tool注册中心
func InitRegistry(e *engine.Engine) *ToolRegistry {
    registry := NewToolRegistry()

    // 游戏会话Tools
    registry.Register(NewNewGameTool(e), []string{"main", "memory"}, "game")
    registry.Register(NewLoadGameTool(e), []string{"main", "memory"}, "game")
    registry.Register(NewSaveGameTool(e), []string{"main", "memory"}, "game")
    registry.Register(NewListGamesTool(e), []string{"main", "memory"}, "game")
    registry.Register(NewDeleteGameTool(e), []string{"main", "memory"}, "game")
    registry.Register(NewGetGameTool(e), []string{"main", "memory"}, "game")

    // 角色管理Tools
    registry.Register(NewCreatePCTool(e), []string{"character", "main"}, "actor")
    registry.Register(NewCreateNPCTool(e), []string{"character", "npc"}, "actor")
    registry.Register(NewCreateEnemyTool(e), []string{"character", "combat", "npc"}, "actor")
    registry.Register(NewCreateCompanionTool(e), []string{"character"}, "actor")
    registry.Register(NewGetActorTool(e), []string{"character", "combat", "rules", "main"}, "actor")
    registry.Register(NewGetPCTool(e), []string{"character", "main"}, "actor")
    registry.Register(NewUpdateActorTool(e), []string{"character"}, "actor")
    registry.Register(NewRemoveActorTool(e), []string{"character"}, "actor")
    registry.Register(NewListActorsTool(e), []string{"character", "main"}, "actor")

    // 升级与经验Tools
    registry.Register(NewAddExperienceTool(e), []string{"character"}, "progression")
    registry.Register(NewLevelUpTool(e), []string{"character"}, "progression")

    // 休息系统Tools
    registry.Register(NewShortRestTool(e), []string{"character"}, "rest")
    registry.Register(NewStartLongRestTool(e), []string{"character"}, "rest")
    registry.Register(NewEndLongRestTool(e), []string{"character"}, "rest")

    // 背景系统Tools
    registry.Register(NewApplyBackgroundTool(e), []string{"character"}, "background")
    registry.Register(NewGetBackgroundFeaturesTool(e), []string{"character"}, "background")

    // 多职业系统Tools
    registry.Register(NewValidateMulticlassChoiceTool(e), []string{"character"}, "multiclass")
    registry.Register(NewGetMulticlassSpellSlotsTool(e), []string{"character", "rules"}, "multiclass")

    // 生活方式Tools
    registry.Register(NewSetLifestyleTool(e), []string{"character"}, "lifestyle")
    registry.Register(NewAdvanceGameTimeTool(e), []string{"character", "memory"}, "lifestyle")

    // 战斗系统Tools
    registry.Register(NewStartCombatTool(e), []string{"combat", "main"}, "combat")
    registry.Register(NewStartCombatWithSurpriseTool(e), []string{"combat", "main"}, "combat")
    registry.Register(NewEndCombatTool(e), []string{"combat", "main"}, "combat")
    registry.Register(NewGetCurrentCombatTool(e), []string{"combat", "main"}, "combat")
    registry.Register(NewNextTurnTool(e), []string{"combat"}, "combat")
    registry.Register(NewGetCurrentTurnTool(e), []string{"combat", "main"}, "combat")
    registry.Register(NewExecuteActionTool(e), []string{"combat"}, "combat")
    registry.Register(NewExecuteAttackTool(e), []string{"combat"}, "combat")
    registry.Register(NewExecuteDamageTool(e), []string{"combat"}, "combat")
    registry.Register(NewExecuteHealingTool(e), []string{"combat"}, "combat")
    registry.Register(NewMoveActorTool(e), []string{"combat", "movement"}, "combat")
    registry.Register(NewAttemptOpportunityAttackTool(e), []string{"combat"}, "combat")

    // 死亡豁免Tools
    registry.Register(NewPerformDeathSaveTool(e), []string{"combat"}, "death_save")
    registry.Register(NewStabilizeCreatureTool(e), []string{"combat"}, "death_save")
    registry.Register(NewGetDeathSaveStatusTool(e), []string{"combat"}, "death_save")

    // 环境系统Tools
    registry.Register(NewSetEnvironmentTool(e), []string{"combat", "narrative"}, "environment")
    registry.Register(NewResolveEnvironmentalDamageTool(e), []string{"combat", "rules"}, "environment")

    // 检定系统Tools
    registry.Register(NewPerformAbilityCheckTool(e), []string{"rules"}, "check")
    registry.Register(NewPerformSkillCheckTool(e), []string{"rules"}, "check")
    registry.Register(NewPerformSavingThrowTool(e), []string{"rules", "combat"}, "check")
    registry.Register(NewGetSkillAbilityTool(e), []string{"rules"}, "check")
    registry.Register(NewGetPassivePerceptionTool(e), []string{"rules", "npc"}, "check")

    // 法术系统Tools
    registry.Register(NewCastSpellTool(e), []string{"rules", "combat"}, "spell")
    registry.Register(NewCastSpellRitualTool(e), []string{"rules"}, "spell")
    registry.Register(NewGetSpellSlotsTool(e), []string{"rules", "character"}, "spell")
    registry.Register(NewPrepareSpellsTool(e), []string{"rules", "character"}, "spell")
    registry.Register(NewLearnSpellTool(e), []string{"rules", "character"}, "spell")
    registry.Register(NewConcentrationCheckTool(e), []string{"rules", "combat"}, "spell")
    registry.Register(NewEndConcentrationTool(e), []string{"rules"}, "spell")
    registry.Register(NewIsConcentratingTool(e), []string{"rules"}, "spell")
    registry.Register(NewGetConcentrationSpellTool(e), []string{"rules"}, "spell")
    registry.Register(NewGetPactMagicSlotsTool(e), []string{"rules", "character"}, "spell")
    registry.Register(NewRestorePactMagicSlotsTool(e), []string{"rules", "character"}, "spell")

    // 库存管理Tools
    registry.Register(NewAddItemTool(e), []string{"inventory", "character"}, "inventory")
    registry.Register(NewRemoveItemTool(e), []string{"inventory", "character"}, "inventory")
    registry.Register(NewGetInventoryTool(e), []string{"inventory", "character"}, "inventory")
    registry.Register(NewEquipItemTool(e), []string{"inventory", "character"}, "inventory")
    registry.Register(NewUnequipItemTool(e), []string{"inventory", "character"}, "inventory")
    registry.Register(NewGetEquipmentTool(e), []string{"inventory", "character"}, "inventory")
    registry.Register(NewAttuneItemTool(e), []string{"inventory", "character"}, "inventory")
    registry.Register(NewTransferItemTool(e), []string{"inventory"}, "inventory")
    registry.Register(NewAddCurrencyTool(e), []string{"inventory", "character"}, "inventory")

    // 魔法物品Tools
    registry.Register(NewUseMagicItemTool(e), []string{"inventory", "combat"}, "magic_item")
    registry.Register(NewUnattuneItemTool(e), []string{"inventory"}, "magic_item")
    registry.Register(NewRechargeMagicItemsTool(e), []string{"inventory"}, "magic_item")
    registry.Register(NewGetMagicItemBonusTool(e), []string{"inventory", "rules"}, "magic_item")

    // 专长系统Tools
    registry.Register(NewSelectFeatTool(e), []string{"character"}, "feat")
    registry.Register(NewListFeatsTool(e), []string{"character", "memory"}, "feat")
    registry.Register(NewGetFeatDetailsTool(e), []string{"character", "memory"}, "feat")
    registry.Register(NewRemoveFeatTool(e), []string{"character"}, "feat")
    registry.Register(NewGetActorFeatsTool(e), []string{"character"}, "feat")

    // 场景管理Tools
    registry.Register(NewCreateSceneTool(e), []string{"narrative"}, "scene")
    registry.Register(NewGetSceneTool(e), []string{"narrative", "main"}, "scene")
    registry.Register(NewUpdateSceneTool(e), []string{"narrative"}, "scene")
    registry.Register(NewDeleteSceneTool(e), []string{"narrative"}, "scene")
    registry.Register(NewListScenesTool(e), []string{"narrative", "main"}, "scene")
    registry.Register(NewSetCurrentSceneTool(e), []string{"narrative"}, "scene")
    registry.Register(NewGetCurrentSceneTool(e), []string{"narrative", "main"}, "scene")
    registry.Register(NewAddSceneConnectionTool(e), []string{"narrative"}, "scene")
    registry.Register(NewRemoveSceneConnectionTool(e), []string{"narrative"}, "scene")
    registry.Register(NewMoveActorToSceneTool(e), []string{"narrative"}, "scene")
    registry.Register(NewGetSceneActorsTool(e), []string{"narrative", "main"}, "scene")
    registry.Register(NewAddItemToSceneTool(e), []string{"narrative"}, "scene")
    registry.Register(NewRemoveItemFromSceneTool(e), []string{"narrative"}, "scene")
    registry.Register(NewGetSceneItemsTool(e), []string{"narrative"}, "scene")

    // 探索系统Tools
    registry.Register(NewStartTravelTool(e), []string{"narrative"}, "exploration")
    registry.Register(NewAdvanceTravelTool(e), []string{"narrative"}, "exploration")
    registry.Register(NewForageTool(e), []string{"narrative", "rules"}, "exploration")
    registry.Register(NewNavigateTool(e), []string{"narrative", "rules"}, "exploration")

    // 社交互动Tools
    registry.Register(NewInteractWithNPCTool(e), []string{"npc"}, "social")
    registry.Register(NewGetNPCAttitudeTool(e), []string{"npc"}, "social")

    // 任务系统Tools
    registry.Register(NewCreateQuestTool(e), []string{"memory"}, "quest")
    registry.Register(NewGetQuestTool(e), []string{"memory", "main"}, "quest")
    registry.Register(NewListQuestsTool(e), []string{"memory", "main"}, "quest")
    registry.Register(NewAcceptQuestTool(e), []string{"memory"}, "quest")
    registry.Register(NewUpdateQuestObjectiveTool(e), []string{"memory"}, "quest")
    registry.Register(NewCompleteQuestTool(e), []string{"memory"}, "quest")
    registry.Register(NewFailQuestTool(e), []string{"memory"}, "quest")
    registry.Register(NewDeleteQuestTool(e), []string{"memory"}, "quest")
    registry.Register(NewGetActorQuestsTool(e), []string{"memory", "main"}, "quest")
    registry.Register(NewGetQuestGiverQuestsTool(e), []string{"memory", "npc"}, "quest")

    // 制作系统Tools
    registry.Register(NewStartCraftingTool(e), []string{"crafting"}, "crafting")
    registry.Register(NewAdvanceCraftingTool(e), []string{"crafting"}, "crafting")
    registry.Register(NewCompleteCraftingTool(e), []string{"crafting"}, "crafting")
    registry.Register(NewGetCraftingRecipesTool(e), []string{"crafting", "memory"}, "crafting")

    // 诅咒系统Tools
    registry.Register(NewCurseActorTool(e), []string{"rules"}, "curse")
    registry.Register(NewRemoveCurseTool(e), []string{"rules"}, "curse")
    registry.Register(NewGetCursesTool(e), []string{"rules"}, "curse")

    // 力竭系统Tools
    registry.Register(NewApplyExhaustionTool(e), []string{"rules"}, "exhaustion")
    registry.Register(NewRemoveExhaustionTool(e), []string{"rules"}, "exhaustion")
    registry.Register(NewGetExhaustionStatusTool(e), []string{"rules"}, "exhaustion")

    // 骑乘系统Tools
    registry.Register(NewMountCreatureTool(e), []string{"mount"}, "mount")
    registry.Register(NewDismountTool(e), []string{"mount"}, "mount")
    registry.Register(NewCalculateMountSpeedTool(e), []string{"mount"}, "mount")

    // 移动系统Tools
    registry.Register(NewPerformJumpTool(e), []string{"movement"}, "movement")
    registry.Register(NewApplyFallDamageTool(e), []string{"movement", "combat"}, "movement")
    registry.Register(NewCalculateBreathHoldingTool(e), []string{"movement"}, "movement")
    registry.Register(NewApplySuffocationTool(e), []string{"movement"}, "movement")
    registry.Register(NewPerformEncounterCheckTool(e), []string{"movement", "narrative"}, "movement")

    // 毒药系统Tools
    registry.Register(NewApplyPoisonTool(e), []string{"rules"}, "poison")
    registry.Register(NewResolvePoisonEffectTool(e), []string{"rules"}, "poison")
    registry.Register(NewRemovePoisonTool(e), []string{"rules"}, "poison")

    // 陷阱系统Tools
    registry.Register(NewPlaceTrapTool(e), []string{"narrative"}, "trap")
    registry.Register(NewDetectTrapTool(e), []string{"narrative", "rules"}, "trap")
    registry.Register(NewDisarmTrapTool(e), []string{"narrative", "rules"}, "trap")
    registry.Register(NewTriggerTrapTool(e), []string{"narrative", "combat"}, "trap")

    // 骰子系统Tools
    registry.Register(NewRollTool(e), []string{"rules"}, "dice")
    registry.Register(NewRollAdvantageTool(e), []string{"rules"}, "dice")
    registry.Register(NewRollDisadvantageTool(e), []string{"rules"}, "dice")
    registry.Register(NewRollAbilityTool(e), []string{"rules"}, "dice")
    registry.Register(NewRollHitDiceTool(e), []string{"character"}, "dice")

    // 状态查询Tools
    registry.Register(NewGetStateSummaryTool(e), []string{"memory", "main"}, "state")
    registry.Register(NewGetActorSheetTool(e), []string{"memory", "character"}, "state")
    registry.Register(NewGetCombatSummaryTool(e), []string{"memory", "combat"}, "state")

    // 阶段管理Tools
    registry.Register(NewSetPhaseTool(e), []string{"memory", "main"}, "phase")
    registry.Register(NewGetPhaseTool(e), []string{"memory", "main"}, "phase")
    registry.Register(NewGetAllowedOperationsTool(e), []string{"memory", "main"}, "phase")

    // 信息聚合查询Tools
    registry.Register(NewGetLifestyleInfoTool(e), []string{"character"}, "info")
    registry.Register(NewGetCraftingInfoTool(e), []string{"crafting"}, "info")
    registry.Register(NewGetCarryingCapacityTool(e), []string{"inventory"}, "info")
    registry.Register(NewGetExhaustionEffectsTool(e), []string{"rules"}, "info")

    // 数据查询Tools (由Data Query Agent统一处理)
    // ListRaces, GetRace, ListClasses, GetClass, etc.

    return registry
}
```

## 6. Tool调用流程

```
Agent发起Tool调用请求
         │
         ▼
┌─────────────────────────┐
│   LLM生成tool_call      │
│   {                     │
│     "name": "xxx",      │
│     "arguments": {...}  │
│   }                     │
└─────────────────────────┘
         │
         ▼
┌─────────────────────────┐
│   ToolRegistry.Get()    │
│   查找对应Tool          │
└─────────────────────────┘
         │
         ▼
┌─────────────────────────┐
│   参数验证              │
│   类型转换              │
└─────────────────────────┘
         │
         ▼
┌─────────────────────────┐
│   Tool.Execute()        │
│   调用引擎API           │
└─────────────────────────┘
         │
         ▼
┌─────────────────────────┐
│   格式化结果            │
│   ToolResult            │
└─────────────────────────┘
         │
         ▼
┌─────────────────────────┐
│   返回给Agent           │
│   作为后续推理的输入    │
└─────────────────────────┘
```
