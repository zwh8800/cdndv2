# 多Agent架构设计

## 1. 概述

本文档描述了 D&D 游戏引擎与 LLM 对接的多Agent架构设计方案。该架构旨在通过多Agent协作模式，让LLM扮演地下城主(DM)角色，同时确保游戏严格遵循D&D 5e规则。

## 2. 设计目标

### 2.1 核心目标

1. **规则合规性**: 所有游戏操作必须严格遵循D&D 5e规则，由引擎强制约束
2. **叙事自由度**: LLM在规则框架内发挥想象力，推进剧情和战斗
3. **复杂度管理**: 通过职责分离降低单个Agent的复杂度
4. **可扩展性**: 支持新增子Agent而不影响现有架构
5. **状态一致性**: 确保游戏状态的原子性和一致性

### 2.2 铁律：禁止游戏逻辑运算

**game_engine 绝不自行运算任何游戏逻辑。**

本模块的唯一职责是整合 D&D 引擎与 LLM，扮演 Agent 调度与交互桥接角色。

| 禁止行为 | 正确做法 |
|---------|----------|
| 自行计算攻击掷骰/伤害 | 调用 `ExecuteAttack` 由引擎计算 |
| 自行计算属性检定/技能检定/豁免 | 调用 `PerformAbilityCheck` / `PerformSkillCheck` / `PerformSavingThrow` |
| 自行计算HP变化/治疗效果 | 调用 `ExecuteDamage` / `ExecuteHealing` |
| 自行计算法术位/法术效果 | 调用 `CastSpell` / `GetSpellSlots` |
| 自行计算先攻/回合顺序 | 调用 `StartCombat` / `NextTurn` |
| 自行计算移动距离/范围 | 调用 `MoveActor` |
| 自行计算升级/经验/熟练加值 | 调用 `AddExperience` / `LevelUp` |
| 自行计算负重/库存/装备效果 | 调用 `AddItem` / `EquipItem` / `GetInventory` |
| 自行计算状态效果/专注/死亡豁免 | 调用引擎对应API |
| 自行实现D&D规则判断逻辑 | 所有规则判断由引擎执行 |

**允许的LLM判断：**
- 决定调用哪个Tool/子Agent
- 选择操作的参数（目标、武器、法术等）
- 根据引擎返回结果生成叙事
- 判断是否需要继续调用Tool
- 引导玩家下一步操作

**简单原则：凡是涉及D&D规则的数值计算、状态变更、规则判定，一律调用引擎。game_engine 只负责调度，不负责运算。**

### 2.3 约束条件

- 仅使用 `dnd-core/pkg/engine` 提供的API
- 所有规则执行由引擎完成，game_engine和LLM不直接计算规则结果
- Tool调用结果由引擎返回，LLM基于结果进行叙事

## 3. 整体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                           玩家交互层                                  │
│                    (接收输入 / 展示输出)                              │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Game Engine                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                      ReAct Loop Controller                     │  │
│  │    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐      │  │
│  │    │   Observe   │───▶│    Think    │───▶│     Act     │      │  │
│  │    └─────────────┘    └─────────────┘    └─────────────┘      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                    │                                 │
│                                    ▼                                 │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                        Main Agent (DM)                         │  │
│  │                                                                │  │
│  │  职责: 剧情推进、决策协调、叙事生成、玩家交互                    │  │
│  │  权限: 调用所有子Agent、直接与玩家对话                          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                    │                                 │
│          ┌─────────────────────────┼─────────────────────────┐     │
│          │                         │                         │     │
│          ▼                         ▼                         ▼     │
│  ┌───────────────┐     ┌───────────────────┐     ┌───────────────┐ │
│  │ Character     │     │ Combat            │     │ Narrative     │ │
│  │ Agent         │     │ Agent             │     │ Agent         │ │
│  │               │     │                   │     │               │ │
│  │ 角色创建/管理 │     │ 战斗流程控制      │     │ 场景叙事/对话 │ │
│  └───────────────┘     └───────────────────┘     └───────────────┘ │
│          │                         │                         │     │
│          ▼                         ▼                         ▼     │
│  ┌───────────────┐     ┌───────────────────┐     ┌───────────────┐ │
│  │ Rules         │     │ NPC              │     │ Memory        │ │
│  │ Agent         │     │ Agent            │     │ Agent         │ │
│  │               │     │                   │     │               │ │
│  │ 规则查询/仲裁 │     │ NPC行为管理      │     │ 长期记忆      │ │
│  └───────────────┘     └───────────────────┘     └───────────────┘ │
│          │                         │                         │     │
│          └─────────────────────────┼─────────────────────────┘     │
│                                    │                                 │
│                                    ▼                                 │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                      Tool Registry                             │  │
│  │         (统一管理所有D&D引擎API的Tool封装)                      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                    │                                 │
└────────────────────────────────────┼─────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        D&D Core Engine                               │
│                  (116个API / 严格规则执行)                           │
└─────────────────────────────────────────────────────────────────────┘
```

## 4. 组件说明

### 4.1 ReAct Loop Controller

ReAct循环控制器是整个游戏引擎的核心调度器，负责：

- **Observe**: 收集当前游戏状态、玩家输入
- **Think**: 调用主Agent进行思考和决策
- **Act**: 执行Agent决策，调用相应的Tools或子Agent

### 4.2 Main Agent (DM Agent)

主Agent扮演DM角色，是系统的核心决策者：

**职责**:
- 接收并理解玩家输入
- 决定需要调用哪些子Agent
- 汇总子Agent返回的信息
- 生成最终叙事输出
- 推进剧情发展

**系统提示词要点**:
- 你是D&D游戏的地下城主
- 你负责创造引人入胜的故事
- 你必须通过Tools与游戏世界交互
- 所有规则判定由引擎执行

### 4.3 Sub-Agents

#### 4.3.1 Character Agent (角色管理Agent)

负责玩家角色的创建和管理：

| 功能 | 相关API |
|------|---------|
| 创建角色 | CreatePC, CreateNPC, CreateEnemy |
| 角色查询 | GetActor, GetPC, ListActors |
| 角色更新 | UpdateActor, RemoveActor |
| 升级管理 | AddExperience, LevelUp |
| 休息恢复 | ShortRest, StartLongRest, EndLongRest |

#### 4.3.2 Combat Agent (战斗Agent)

负责战斗流程控制：

| 功能 | 相关API |
|------|---------|
| 战斗初始化 | StartCombat, StartCombatWithSurprise |
| 回合管理 | NextTurn, GetCurrentTurn |
| 动作执行 | ExecuteAction, ExecuteAttack |
| 伤害治疗 | ExecuteDamage, ExecuteHealing |
| 移动控制 | MoveActor |
| 战斗结束 | EndCombat, GetCurrentCombat |

#### 4.3.3 Narrative Agent (叙事Agent)

负责场景叙事和对话生成：

| 功能 | 相关API |
|------|---------|
| 场景管理 | CreateScene, GetScene, UpdateScene |
| 场景切换 | SetCurrentScene, MoveActorToScene |
| 物品交互 | AddItemToScene, GetSceneItems |
| 场景连接 | AddSceneConnection, RemoveSceneConnection |

#### 4.3.4 Rules Agent (规则Agent)

负责规则查询和仲裁：

| 功能 | 相关API |
|------|---------|
| 属性检定 | PerformAbilityCheck |
| 技能检定 | PerformSkillCheck |
| 豁免检定 | PerformSavingThrow |
| 被动感知 | GetPassivePerception |
| 法术相关 | CastSpell, GetSpellSlots, ConcentrationCheck |

#### 4.3.5 NPC Agent (NPC行为Agent)

负责NPC和敌人的行为管理：

| 功能 | 相关API |
|------|---------|
| NPC交互 | InteractWithNPC, GetNPCAttitude |
| 怪物管理 | CreateEnemyFromStatBlock |
| 怪物动作 | GetMonsterActions, UseLegendaryAction |

#### 4.3.6 Memory Agent (记忆Agent)

负责长期记忆和任务管理：

| 功能 | 相关API |
|------|---------|
| 任务管理 | CreateQuest, UpdateQuestObjective, CompleteQuest |
| 任务查询 | GetQuest, ListQuests, GetActorQuests |
| 游戏存档 | SaveGame, LoadGame, ListGames |

### 4.4 Tool Registry

Tool注册中心统一管理所有D&D引擎API的Tool封装：

- 为每个引擎API创建对应的Tool定义
- 管理Tool的Schema定义
- 处理Tool调用到引擎API的转换
- 格式化引擎返回结果

## 5. 数据流

### 5.1 玩家输入处理流程

```
玩家输入
    │
    ▼
┌─────────────┐
│   Input     │  解析玩家意图
│   Parser    │
└─────────────┘
    │
    ▼
┌─────────────┐
│   Main      │  分析需要哪些操作
│   Agent     │
└─────────────┘
    │
    ├──▶ Character Agent (如果涉及角色)
    │         │
    │         ▼
    │    Tool Calls ──▶ Engine APIs
    │         │
    │         ▼
    │    返回结果
    │
    ├──▶ Combat Agent (如果涉及战斗)
    │         │
    │         ▼
    │    Tool Calls ──▶ Engine APIs
    │         │
    │         ▼
    │    返回结果
    │
    └──▶ 其他子Agent...
              │
              ▼
         汇总结果
              │
              ▼
    ┌─────────────┐
    │   Main      │  生成叙事响应
    │   Agent     │
    └─────────────┘
              │
              ▼
         输出给玩家
```

### 5.2 Tool调用流程

```
Agent决策需要调用Tool
         │
         ▼
┌─────────────────────┐
│   Tool Registry     │
│   查找Tool定义      │
└─────────────────────┘
         │
         ▼
┌─────────────────────┐
│   参数验证          │
│   类型转换          │
└─────────────────────┘
         │
         ▼
┌─────────────────────┐
│   D&D Engine API    │
│   执行规则逻辑      │
└─────────────────────┘
         │
         ▼
┌─────────────────────┐
│   结果格式化        │
│   返回给Agent       │
└─────────────────────┘
```

## 6. 状态管理

### 6.1 游戏状态

游戏状态由引擎完全管理，Agent只通过API访问：

```
GameState
├── GameID: 游戏会话ID
├── CurrentSceneID: 当前场景ID
├── CombatState: 战斗状态(如果正在战斗)
├── Actors: 所有角色列表
├── Scenes: 所有场景列表
├── Quests: 所有任务列表
└── GameTime: 游戏时间
```

### 6.2 Agent上下文

每个Agent调用时传递的上下文信息：

```go
type AgentContext struct {
    GameID      model.ID      // 当前游戏ID
    PlayerID    model.ID      // 当前玩家角色ID
    History     []Message     // 对话历史
    CurrentState *StateSummary // 当前状态摘要
}
```

## 7. 错误处理

### 7.1 引擎错误

引擎返回的错误类型：

```go
var (
    ErrNotFound              // 实体不存在
    ErrAlreadyExists         // 实体已存在
    ErrInvalidState          // 无效状态
    ErrCombatNotActive       // 战斗未激活
    ErrCombatAlreadyActive   // 战斗已激活
    ErrNotYourTurn           // 不是该角色回合
    ErrActionAlreadyUsed     // 动作已使用
    ErrInsufficientSlots     // 法术位不足
    ErrInvalidTarget         // 无效目标
    ErrOutOfRange            // 超出范围
    ErrNoLineOfSight         // 无视线
    ErrConcentrationBroken   // 专注失败
    ErrActorIncapacitated    // 角色失去行动能力
)
```

### 7.2 错误处理策略

1. **可恢复错误**: 返回错误信息给Agent，让Agent调整策略
2. **不可恢复错误**: 终止当前操作，返回错误给玩家
3. **规则违规**: 返回规则说明，引导Agent/玩家正确操作

## 8. 扩展性设计

### 8.1 新增子Agent

新增子Agent需要实现以下接口：

```go
type SubAgent interface {
    // Agent标识
    Name() string

    // Agent可用的Tools
    Tools() []Tool

    // Agent的系统提示词
    SystemPrompt(ctx *AgentContext) string

    // 处理引擎错误
    HandleError(err error) *AgentResponse
}
```

### 8.2 新增Tool

新增Tool需要：

1. 定义Tool Schema
2. 实现参数解析
3. 调用对应引擎API
4. 格式化返回结果

## 9. 目录结构

```
game_engine/
├── engine.go              # 主引擎入口
├── react_loop.go          # ReAct循环控制器
├── context.go             # 上下文管理
│
├── agent/
│   ├── agent.go           # Agent接口定义
│   ├── main_agent.go      # 主Agent(DM)
│   ├── character_agent.go # 角色管理Agent
│   ├── combat_agent.go    # 战斗Agent
│   ├── narrative_agent.go # 叙事Agent
│   ├── rules_agent.go     # 规则Agent
│   ├── npc_agent.go       # NPC行为Agent
│   └── memory_agent.go    # 记忆Agent
│
├── tool/
│   ├── registry.go        # Tool注册中心
│   ├── tool.go            # Tool接口定义
│   ├── game_tools.go      # 游戏会话相关Tools
│   ├── actor_tools.go     # 角色管理Tools
│   ├── combat_tools.go    # 战斗系统Tools
│   ├── spell_tools.go     # 法术系统Tools
│   ├── check_tools.go     # 检定系统Tools
│   ├── inventory_tools.go # 库存管理Tools
│   ├── scene_tools.go     # 场景管理Tools
│   ├── quest_tools.go     # 任务系统Tools
│   └── exploration_tools.go # 探索系统Tools
│
├── llm/
│   ├── client.go          # LLM客户端接口
│   ├── message.go         # 消息格式定义
│   └── response.go        # 响应解析
│
├── prompt/
│   ├── templates.go       # 提示词模板
│   ├── main_system.md     # 主Agent系统提示词
│   └── sub_systems/       # 子Agent系统提示词
│
└── state/
    ├── summary.go         # 状态摘要生成
    └── formatter.go       # 状态格式化
```

## 10. 实现优先级

### Phase 1: 核心框架
1. Tool Registry 基础框架
2. Main Agent 基础实现
3. ReAct Loop 控制器
4. LLM客户端接口

### Phase 2: 核心功能
1. Character Agent + 角色相关Tools
2. Combat Agent + 战斗相关Tools
3. Rules Agent + 检定相关Tools

### Phase 3: 扩展功能
1. Narrative Agent + 场景相关Tools
2. NPC Agent + NPC相关Tools
3. Memory Agent + 任务/存档Tools

### Phase 4: 优化完善
1. 错误处理优化
2. 性能优化
3. 提示词优化
