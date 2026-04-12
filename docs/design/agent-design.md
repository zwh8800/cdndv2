# Agent系统详细设计

## 0. 核心原则

> **game_engine 绝不自行运算任何游戏逻辑。**
> 所有D&D规则运算（掷骰、伤害、检定、法术、移动、状态等）必须调用dnd引擎。
> game_engine 仅负责：①调度Agent ②调用引擎Tool ③让LLM做决策判断 ④生成叙事输出。

## 1. Agent接口定义

### 1.1 基础接口

```go
// Agent 基础接口
type Agent interface {
    // Name 返回Agent名称
    Name() string

    // Description 返回Agent描述
    Description() string

    // SystemPrompt 返回系统提示词
    SystemPrompt(ctx *AgentContext) string

    // Tools 返回Agent可用的Tools
    Tools() []Tool

    // Execute 执行Agent逻辑
    Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
}

// AgentContext Agent执行上下文
type AgentContext struct {
    GameID       model.ID        // 游戏会话ID
    PlayerID     model.ID        // 玩家角色ID
    Engine       *engine.Engine  // D&D引擎实例
    History      []Message       // 对话历史
    CurrentState *StateSummary   // 当前状态摘要
    Metadata     map[string]any  // 扩展元数据
}

// AgentRequest Agent请求
type AgentRequest struct {
    UserInput    string          // 用户输入
    Intent       Intent          // 解析后的意图
    Context      *AgentContext   // 执行上下文
    SubAgentResults map[string]*AgentResponse // 子Agent返回结果
}

// AgentResponse Agent响应
type AgentResponse struct {
    Content     string          // 生成的文本内容
    ToolCalls   []ToolCall      // 需要执行的Tool调用
    NextAction  NextAction      // 下一步动作
    StateChange *StateChange    // 状态变更
    Errors      []AgentError    // 错误信息
}

// NextAction 下一步动作类型
type NextAction int

const (
    ActionContinue    NextAction = iota  // 继续思考
    ActionCallSubAgent                    // 调用子Agent
    ActionRespondToPlayer                 // 响应玩家
    ActionWaitForInput                    // 等待玩家输入
    ActionEndGame                         // 结束游戏
)
```

### 1.2 子Agent接口

```go
// SubAgent 子Agent接口
type SubAgent interface {
    Agent

    // CanHandle 判断是否能处理该意图
    CanHandle(intent Intent) bool

    // Priority 返回处理优先级
    Priority() int

    // Dependencies 返回依赖的其他Agent
    Dependencies() []string
}
```

## 2. Main Agent (DM Agent)

### 2.1 职责定义

主Agent是整个系统的核心决策者，扮演地下城主(DM)角色：

| 职责 | 说明 |
|------|------|
| 意图理解 | 解析玩家输入，理解玩家意图 |
| 任务分解 | 将复杂任务分解为子任务 |
| Agent调度 | 决定调用哪些子Agent，以及调用顺序 |
| 信息汇总 | 汇总子Agent返回的结果 |
| 叙事生成 | 基于规则结果生成引人入胜的叙事 |
| 玩家引导 | 引导玩家进行下一步操作 |

### 2.2 系统提示词

```markdown
# 角色定义

你是一位经验丰富的地下城主(Dungeon Master)，负责主持一场D&D 5e游戏。
你的职责是创造引人入胜的故事、管理游戏世界、引导玩家冒险。

# 核心原则

1. **规则至上**: 所有规则判定必须通过调用Tools完成，你不得自行计算
2. **叙事驱动**: 在规则框架内发挥想象力，创造精彩的故事
3. **玩家中心**: 关注玩家体验，提供清晰的选择和引导
4. **公平公正**: 按照规则执行，不偏袒任何一方

# 可用能力

你可以调用以下子Agent来处理特定任务：
- `character_agent`: 角色创建、属性管理、升级、休息
- `combat_agent`: 战斗流程、回合管理、攻击伤害
- `narrative_agent`: 场景管理、物品交互、环境描述
- `rules_agent`: 检定、豁免、法术施放
- `npc_agent`: NPC行为、态度、互动
- `memory_agent`: 任务管理、游戏存档

# 工作流程

1. 分析玩家输入，理解意图
2. 判断需要调用哪些子Agent
3. 等待子Agent返回结果
4. 基于结果生成叙事响应
5. 引导玩家下一步行动

# 输出格式

你的输出应该：
- 使用生动的语言描述场景和行动
- 清晰地传达规则结果（如检定结果、伤害数值）
- 提供玩家可选择的行动选项
- 保持故事连贯性和节奏感

# 禁止行为

- 自行计算任何规则数值（必须通过Tool调用）
- 忽略或绕过规则限制
- 强制玩家做出特定选择
- 泄露玩家不应知道的信息
```

### 2.3 Tool分配

主Agent拥有调用子Agent的特殊Tool：

```go
var mainAgentTools = []Tool{
    &CallSubAgentTool{Name: "call_character_agent"},
    &CallSubAgentTool{Name: "call_combat_agent"},
    &CallSubAgentTool{Name: "call_narrative_agent"},
    &CallSubAgentTool{Name: "call_rules_agent"},
    &CallSubAgentTool{Name: "call_npc_agent"},
    &CallSubAgentTool{Name: "call_memory_agent"},
    &GetGameStateTool{},      // 获取游戏状态摘要
    &EndTurnTool{},           // 结束当前回合
}
```

### 2.4 决策流程

```
┌─────────────────────────────────────────────────────────────────┐
│                      Main Agent 决策流程                         │
└─────────────────────────────────────────────────────────────────┘

玩家输入: "我想攻击那只哥布林"
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. 意图分析                                                      │
│    - 输入类型: 战斗动作                                          │
│    - 涉及实体: 玩家角色、哥布林                                   │
│    - 所需子系统: combat_agent                                    │
└─────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. 状态检查                                                      │
│    - 是否在战斗中? ──调用 GetGameStateTool                       │
│    - 当前是否玩家回合?                                           │
│    - 玩家是否在攻击范围内?                                       │
└─────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. 调用子Agent                                                    │
│    call_combat_agent({                                           │
│        "action": "attack",                                       │
│        "target": "goblin_001",                                   │
│        "weapon": "longsword"                                     │
│    })                                                            │
└─────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. 子Agent返回                                                    │
│    {                                                             │
│        "attack_roll": 15,                                        │
│        "target_ac": 13,                                          │
│        "hit": true,                                              │
│        "damage": 8,                                              │
│        "goblin_hp_remaining": 0,                                 │
│        "goblin_defeated": true                                   │
│    }                                                             │
└─────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. 生成叙事响应                                                   │
│                                                                  │
│    "你挥舞长剑，剑刃划破空气直奔哥布林！                          │
│     攻击掷骰15，命中！(哥布林AC 13)                               │
│     你的剑深深刺入哥布林的胸膛，造成8点伤害！                     │
│     哥布林发出一声惨叫，倒地不起。"                               │
└─────────────────────────────────────────────────────────────────┘
```

## 3. Character Agent

### 3.1 职责定义

| 职责 | 说明 |
|------|------|
| 角色创建 | 创建PC、NPC、敌人、同伴 |
| 属性管理 | 查询和更新角色属性 |
| 升级处理 | 经验值添加、等级提升 |
| 休息恢复 | 短休、长休的生命值和资源恢复 |
| 装备管理 | 物品获取、装备穿戴、调谐 |

### 3.2 系统提示词

```markdown
# 角色定义

你是D&D 5e角色管理专家，负责处理所有与角色相关的操作。
你熟悉D&D 5e的角色创建规则、升级机制和资源管理。

# 核心原则

1. 所有角色操作必须通过Tool调用引擎API
2. 严格遵守D&D 5e规则
3. 提供清晰的角色状态信息

# 可用Tools

## 角色创建
- create_pc: 创建玩家角色
- create_npc: 创建NPC
- create_enemy: 创建敌人
- create_companion: 创建同伴

## 角色查询
- get_actor: 获取角色基本信息
- get_pc: 获取玩家角色详情
- list_actors: 列出所有角色

## 角色更新
- update_actor: 更新角色状态
- remove_actor: 移除角色

## 升级与经验
- add_experience: 添加经验值
- level_up: 角色升级

## 休息系统
- short_rest: 执行短休
- start_long_rest: 开始长休
- end_long_rest: 结束长休

## 库存管理
- add_item: 添加物品
- remove_item: 移除物品
- get_inventory: 获取库存
- equip_item: 装备物品
- unequip_item: 卸下装备
- attune_item: 调谐魔法物品

# 输出格式

返回结构化的角色操作结果，包括：
- 操作是否成功
- 角色状态变更
- 相关数值变化
```

### 3.3 Tool定义

```go
// 角色创建Tools
var characterCreationTools = []Tool{
    &CreatePCTool{},
    &CreateNPCTool{},
    &CreateEnemyTool{},
    &CreateCompanionTool{},
}

// 角色查询Tools
var characterQueryTools = []Tool{
    &GetActorTool{},
    &GetPCTool{},
    &ListActorsTool{},
    &GetInventoryTool{},
    &GetEquipmentTool{},
}

// 角色更新Tools
var characterUpdateTools = []Tool{
    &UpdateActorTool{},
    &RemoveActorTool{},
    &AddExperienceTool{},
    &LevelUpTool{},
}

// 休息Tools
var restTools = []Tool{
    &ShortRestTool{},
    &StartLongRestTool{},
    &EndLongRestTool{},
}

// 库存Tools
var inventoryTools = []Tool{
    &AddItemTool{},
    &RemoveItemTool{},
    &EquipItemTool{},
    &UnequipItemTool{},
    &AttuneItemTool{},
    &TransferItemTool{},
    &AddCurrencyTool{},
}
```

### 3.4 典型场景

#### 创建角色

```json
// Tool调用
{
    "tool": "create_pc",
    "arguments": {
        "name": "艾尔文",
        "race": "elf",
        "classes": [{"class": "fighter", "level": 1}],
        "ability_scores": {
            "strength": 16,
            "dexterity": 14,
            "constitution": 14,
            "intelligence": 10,
            "wisdom": 12,
            "charisma": 8
        },
        "background": "soldier"
    }
}

// 返回结果
{
    "success": true,
    "actor_id": "pc_001",
    "message": "成功创建1级精灵战士艾尔文",
    "details": {
        "hit_points": 12,
        "armor_class": 16,
        "speed": 30,
        "proficiency_bonus": 2,
        "saving_throws": ["strength", "constitution"]
    }
}
```

## 4. Combat Agent

### 4.1 职责定义

| 职责 | 说明 |
|------|------|
| 战斗初始化 | 开始战斗、计算先攻、处理突袭 |
| 回合管理 | 控制回合顺序、推进回合 |
| 动作执行 | 攻击、施法、移动、其他动作 |
| 伤害处理 | 伤害计算、治疗、死亡豁免 |
| 战斗结束 | 判断战斗结束、发放经验 |

### 4.2 系统提示词

```markdown
# 角色定义

你是D&D 5e战斗系统专家，负责管理战斗流程和执行战斗动作。
你熟悉所有战斗规则，包括动作类型、攻击判定、伤害计算等。

# 核心原则

1. 严格按照先攻顺序执行
2. 每个角色每回合只能执行规定数量的动作
3. 所有攻击和伤害由引擎计算
4. 正确处理特殊状态（倒地、束缚等）

# 可用Tools

## 战斗初始化
- start_combat: 开始战斗
- start_combat_with_surprise: 带突袭的战斗

## 回合管理
- get_current_combat: 获取当前战斗状态
- next_turn: 推进到下一回合
- get_current_turn: 获取当前回合信息

## 动作执行
- execute_action: 执行动作（冲刺、脱离、闪避等）
- execute_attack: 执行攻击
- move_actor: 移动角色

## 伤害与治疗
- execute_damage: 施加伤害
- execute_healing: 治疗
- perform_death_save: 死亡豁免
- stabilize_creature: 稳定濒死生物

## 战斗结束
- end_combat: 结束战斗

# 战斗流程

1. 确认战斗开始条件
2. 掷先攻并排序
3. 按先攻顺序处理每回合
4. 执行角色动作
5. 判断战斗结束条件
6. 战后处理（经验、战利品）
```

### 4.3 Tool定义

```go
// 战斗初始化Tools
var combatInitTools = []Tool{
    &StartCombatTool{},
    &StartCombatWithSurpriseTool{},
}

// 回合管理Tools
var turnManagementTools = []Tool{
    &GetCurrentCombatTool{},
    &NextTurnTool{},
    &GetCurrentTurnTool{},
}

// 动作执行Tools
var actionExecutionTools = []Tool{
    &ExecuteActionTool{},
    &ExecuteAttackTool{},
    &MoveActorTool{},
    &AttemptOpportunityAttackTool{},
}

// 伤害治疗Tools
var damageHealingTools = []Tool{
    &ExecuteDamageTool{},
    &ExecuteHealingTool{},
    &PerformDeathSaveTool{},
    &StabilizeCreatureTool{},
    &GetDeathSaveStatusTool{},
}
```

### 4.4 战斗状态机

```
┌─────────────────────────────────────────────────────────────────┐
│                        战斗状态机                                │
└─────────────────────────────────────────────────────────────────┘

     ┌──────────┐
     │  IDLE    │ ← 非战斗状态
     └────┬─────┘
          │ start_combat
          ▼
     ┌──────────┐
     │ ROLLING  │ ← 先攻掷骰
     │INITIATIVE│
     └────┬─────┘
          │ 排序完成
          ▼
     ┌──────────┐
     │  ROUND   │ ← 回合循环
     │  START   │
     └────┬─────┘
          │
          ▼
     ┌──────────┐
     │   TURN   │ ← 当前回合
     │  START   │
     └────┬─────┘
          │
          ├──────────────────────┐
          │                      │
          ▼                      ▼
     ┌──────────┐          ┌──────────┐
     │  ACTION  │          │  BONUS   │
     │  PHASE   │          │  ACTION  │
     └────┬─────┘          └────┬─────┘
          │                      │
          └──────────┬───────────┘
                     │
                     ▼
              ┌──────────┐
              │ MOVEMENT │
              │  PHASE   │
              └────┬─────┘
                   │
                   ▼
              ┌──────────┐
              │   TURN   │
              │   END    │
              └────┬─────┘
                   │
          ┌────────┴────────┐
          │                 │
          ▼                 ▼
     ┌──────────┐     ┌──────────┐
     │  NEXT    │     │  COMBAT  │
     │  TURN    │     │   END    │
     └──────────┘     └──────────┘
```

## 5. Rules Agent

### 5.1 职责定义

| 职责 | 说明 |
|------|------|
| 属性检定 | 执行力量、敏捷等属性检定 |
| 技能检定 | 执行各种技能检定 |
| 豁免检定 | 执行豁免检定 |
| 法术系统 | 施法、专注检定、法术位管理 |

### 5.2 系统提示词

```markdown
# 角色定义

你是D&D 5e规则仲裁专家，负责执行各种检定和法术操作。
你精通所有检定规则、难度等级设定和法术系统。

# 核心原则

1. 所有检定由引擎执行，只提供参数
2. 正确设置DC难度
3. 处理优势/劣势情况
4. 法术需要消耗法术位

# 可用Tools

## 检定系统
- perform_ability_check: 属性检定
- perform_skill_check: 技能检定
- perform_saving_throw: 豁免检定
- get_passive_perception: 被动感知

## 法术系统
- cast_spell: 施放法术
- cast_spell_ritual: 仪式施法
- get_spell_slots: 获取法术位
- prepare_spells: 准备法术
- learn_spell: 学习法术
- concentration_check: 专注检定
- end_concentration: 结束专注
- is_concentrating: 检查专注状态

# DC难度参考

- 5: 非常简单
- 10: 简单
- 15: 中等
- 20: 困难
- 25: 非常困难
- 30: 几乎不可能
```

### 5.3 Tool定义

```go
// 检定Tools
var checkTools = []Tool{
    &PerformAbilityCheckTool{},
    &PerformSkillCheckTool{},
    &PerformSavingThrowTool{},
    &GetPassivePerceptionTool{},
    &GetSkillAbilityTool{},
}

// 法术Tools
var spellTools = []Tool{
    &CastSpellTool{},
    &CastSpellRitualTool{},
    &GetSpellSlotsTool{},
    &PrepareSpellsTool{},
    &LearnSpellTool{},
    &ConcentrationCheckTool{},
    &EndConcentrationTool{},
    &IsConcentratingTool{},
    &GetConcentrationSpellTool{},
    &GetPactMagicSlotsTool{},
    &RestorePactMagicSlotsTool{},
}
```

## 6. Narrative Agent

### 6.1 职责定义

| 职责 | 说明 |
|------|------|
| 场景管理 | 创建、查询、更新、删除场景 |
| 场景切换 | 角色在场景间移动 |
| 环境交互 | 场景内物品、陷阱等 |
| 探索系统 | 旅行、觅食、导航 |

### 6.2 系统提示词

```markdown
# 角色定义

你是D&D 5e场景管理专家，负责管理游戏世界中的场景和探索。

# 可用Tools

## 场景管理
- create_scene: 创建场景
- get_scene: 获取场景信息
- update_scene: 更新场景
- delete_scene: 删除场景
- list_scenes: 列出所有场景

## 场景导航
- set_current_scene: 设置当前场景
- get_current_scene: 获取当前场景
- add_scene_connection: 添加场景连接
- remove_scene_connection: 移除场景连接

## 角色位置
- move_actor_to_scene: 移动角色到另一场景
- get_scene_actors: 获取场景内角色

## 场景物品
- add_item_to_scene: 添加物品到场景
- remove_item_from_scene: 从场景移除物品
- get_scene_items: 获取场景物品

## 探索系统
- start_travel: 开始旅行
- advance_travel: 推进旅行
- forage: 觅食
- navigate: 导航检定
```

### 6.3 Tool定义

```go
var narrativeTools = []Tool{
    // 场景管理
    &CreateSceneTool{},
    &GetSceneTool{},
    &UpdateSceneTool{},
    &DeleteSceneTool{},
    &ListScenesTool{},

    // 场景导航
    &SetCurrentSceneTool{},
    &GetCurrentSceneTool{},
    &AddSceneConnectionTool{},
    &RemoveSceneConnectionTool{},

    // 角色位置
    &MoveActorToSceneTool{},
    &GetSceneActorsTool{},

    // 场景物品
    &AddItemToSceneTool{},
    &RemoveItemFromSceneTool{},
    &GetSceneItemsTool{},

    // 探索
    &StartTravelTool{},
    &AdvanceTravelTool{},
    &ForageTool{},
    &NavigateTool{},
}
```

## 7. NPC Agent

### 7.1 职责定义

| 职责 | 说明 |
|------|------|
| NPC互动 | 处理玩家与NPC的社交互动 |
| 态度管理 | 管理NPC对玩家的态度 |
| 怪物行为 | 处理敌人的战斗行为 |

### 7.2 系统提示词

```markdown
# 角色定义

你是D&D 5e NPC行为专家，负责管理NPC和怪物的行为。

# 可用Tools

## 社交互动
- interact_with_npc: 与NPC互动
- get_npc_attitude: 获取NPC态度

## 怪物管理
- create_enemy_from_stat_block: 从数据块创建怪物
- get_monster_actions: 获取怪物可用动作
- use_legendary_action: 使用传说动作
- use_recharge_action: 使用充能动作
- recharge_monster_actions: 充能怪物动作
```

### 7.3 Tool定义

```go
var npcTools = []Tool{
    // 社交互动
    &InteractWithNPCTool{},
    &GetNPCAttitudeTool{},

    // 怪物管理
    &CreateEnemyFromStatBlockTool{},
    &GetMonsterActionsTool{},
    &UseLegendaryActionTool{},
    &UseRechargeActionTool{},
    &RechargeMonsterActionsTool{},
}
```

## 8. Memory Agent

### 8.1 职责定义

| 职责 | 说明 |
|------|------|
| 任务管理 | 创建、更新、完成、失败任务 |
| 游戏存档 | 保存、加载、列出游戏 |
| 专长管理 | 获取、选择专长 |

### 8.2 系统提示词

```markdown
# 角色定义

你是D&D 5e记忆管理专家，负责管理任务、存档和专长。

# 可用Tools

## 任务系统
- create_quest: 创建任务
- get_quest: 获取任务信息
- list_quests: 列出所有任务
- accept_quest: 接受任务
- update_quest_objective: 更新任务目标
- complete_quest: 完成任务
- fail_quest: 任务失败
- get_actor_quests: 获取角色任务
- get_quest_giver_quests: 获取NPC发布的任务

## 游戏存档
- save_game: 保存游戏
- load_game: 加载游戏
- list_games: 列出存档
- delete_game: 删除存档

## 专长系统
- list_feats: 列出可选专长
- get_feat_details: 获取专长详情
- select_feat: 选择专长
- remove_feat: 移除专长
- get_actor_feats: 获取角色专长
```

### 8.3 Tool定义

```go
var memoryTools = []Tool{
    // 任务系统
    &CreateQuestTool{},
    &GetQuestTool{},
    &ListQuestsTool{},
    &AcceptQuestTool{},
    &UpdateQuestObjectiveTool{},
    &CompleteQuestTool{},
    &FailQuestTool{},
    &GetActorQuestsTool{},
    &GetQuestGiverQuestsTool{},

    // 游戏存档
    &SaveGameTool{},
    &LoadGameTool{},
    &ListGamesTool{},
    &DeleteGameTool{},

    // 专长系统
    &ListFeatsTool{},
    &GetFeatDetailsTool{},
    &SelectFeatTool{},
    &RemoveFeatTool{},
    &GetActorFeatsTool{},
}
```

## 9. Agent协作模式

### 9.1 单Agent处理

简单请求由单个子Agent处理：

```
玩家: "我想做一个力量检定推开门"
     │
     ▼
Main Agent 识别意图 → 调用 Rules Agent
                              │
                              ▼
                    Rules Agent 调用 perform_ability_check
                              │
                              ▼
                    返回结果给 Main Agent
                              │
                              ▼
                    Main Agent 生成叙事响应
```

### 9.2 多Agent协作

复杂请求需要多个子Agent协作：

```
玩家: "我想攻击哥布林然后移动到门口"
     │
     ▼
Main Agent 分解任务
     │
     ├─→ Combat Agent (攻击)
     │        │
     │        └─→ 返回攻击结果
     │
     └─→ Narrative Agent (移动)
              │
              └─→ 返回移动结果

Main Agent 汇总结果 → 生成响应
```

### 9.3 Agent依赖处理

某些操作有依赖关系：

```
玩家: "我想对哥布林施放火球术"

需要:
1. Rules Agent: 检查法术位
2. Rules Agent: 施放法术
3. Combat Agent: 处理伤害
4. Narrative Agent: 更新场景状态(如果有环境效果)

执行顺序:
Rules Agent → Combat Agent → Narrative Agent
```
