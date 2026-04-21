# cdndv2 架构改进设计方案

> 基于行业最佳实践与当前架构分析，针对 LLM Tool Calling 遗漏、误用、重复调用问题的全面改进方案
>
> **最后更新**: 基于 `refactor-sub-agent` 分支最新代码（含上下文压缩器、Phase 工具）分析更新

---

## 目录

1. [问题诊断](#1-问题诊断)
2. [核心设计原则](#2-核心设计原则)
3. [改进方案总览](#3-改进方案总览)
4. [P0: 动态 Tool 过滤与上下文裁剪](#4-p0-动态-tool-过滤与上下文裁剪)
5. [P0: 亚代理工具隔离与描述增强](#5-p0-亚代理工具隔离与描述增强)
6. [P0: 复合工具（Composite Tools）合并细粒度 API](#6-p0-复合工具composite-tools合并细粒度-api)
7. [P1: 状态引用机制（State IDs）与上下文自动注入](#7-p1-状态引用机制state-ids与上下文自动注入)
8. [P1: 游戏状态摘要增强](#8-p1-游戏状态摘要增强)
9. [P1: ReAct 循环改进——反射检测与计划优先](#9-p1-react-循环改进反射检测与计划优先)
10. [P2: 路由层统一与代理覆盖补全](#10-p2-路由层统一与代理覆盖补全)
11. [P2: 持久化记忆与反思机制](#11-p2-持久化记忆与反思机制)
12. [P2: 验证代理（Verification Agent）](#12-p2-验证代理verification-agent)
13. [已有的改进与差距分析](#13-已有的改进与差距分析)
14. [实施路线图](#14-实施路线图)
15. [参考资料](#15-参考资料)

---

## 1. 问题诊断

### 1.1 核心症状

| 症状 | 表现 |
|------|------|
| **工具遗漏** | LLM 在应使用特定工具时跳过，直接生成文本回答 |
| **工具误用** | 调用了错误的工具或填入错误参数（如用名字代替 ID） |
| **重复调用** | 相同工具在同一 ReAct 周期内被反复调用 |
| **参数错误** | `actor_id`、`scene_id` 等参数经常传入名字而非 ID |
| **延迟过高** | 单次用户输入触发 3-12+ 次 LLM 调用 |

### 1.2 根因分析

对项目源码的深入分析揭示以下结构性问题：

#### P1: Tool Schema 爆炸

```
当前状态: MainAgent 每次调用发送 36 个 Tool Schema (35 readOnly + delegate_task)
SubAgent 每次调用发送 9-15 个 Tool Schema
```

OpenAI 官方建议每次 LLM 调用不超过 **10-20 个 tool**，超过 10 个后准确率显著下降。当前 36 个 schema 已远超阈值。

#### P2: Tool 描述双重冗余

`main_system.md` 在系统提示中**内联列出所有 readOnly 工具名称和描述**，同时 `GetReadOnlySchemas()` 又将相同的工具作为 function-calling schema 发送。同一条信息被发送两遍，浪费 token。

#### P3: 路由/委派覆盖缺口

`delegate_task` 工具的 `agent_name` 枚举仅包含 3 个代理：`character_agent`、`combat_agent`、`rules_agent`。而架构中实际存在 **11 个 SubAgent**，以下 8 个代理完全无法被主代理委派到达：

- `narrative_agent` — 场景管理、探索、旅行
- `inventory_agent` — 物品、装备、货币
- `npc_agent` — NPC 交互
- `memory_agent` — 任务、生活方式、时间推进
- `movement_agent` — 跳跃、坠落、窒息、遭遇检定
- `mount_agent` — 骑乘/下骑
- `crafting_agent` — 制作系统
- `data_query_agent` — 规则数据查询

同样的问题存在于 `RouterAgent.route_decision` 函数定义中。

#### P4: 无上下文感知的 Tool 过滤

虽然项目已引入 Phase 概念（`set_phase`/`get_phase` 工具，`react_loop.go` 中自动从 `character_creation` 推进到 `exploration`），但 **Tool 过滤尚未基于 Phase 实现**。所有 readOnly 工具仍然每次全量发送：
- 角色创建阶段：暴露战斗工具（无意义）
- 战斗进行中：暴露场景创建工具（极低概率使用）
- 探索阶段：暴露死亡豁免工具（不相关）

Phase 基础设施已就位（Phase 枚举、自动推进、状态摘要中包含 `Phase` 字段），但尚未与 Tool Registry 联动。

#### P5: SubAgent 不区分读写

`base_sub_agent.go:103` 通过 `registry.GetByAgent()` 获取代理的全部工具，**不区分 readOnly 和 write**。SubAgent 的 mini-ReAct 循环每轮都携带全部工具。

#### P6: 游戏状态摘要过于稀疏

`CollectSummary()` 仅返回：
- 游戏名 + 阶段
- 场景名 + 描述
- 玩家：名字、HP/Max、AC
- 战斗：回合数 + 当前行动者
- 活跃任务标题

缺少：法术位、状态效果、物品栏、装备、能力值、技能修正值等。LLM 必须额外调用工具来获取这些信息，增加了调用次数和延迟。

#### P7: Entity ID 传播机制脆弱

`KnownEntityIDs` 仅处理 `actor_id` 和 `scene_id` 两种 ID 类型，LLM 经常将角色名而非 ID 传入参数，且当 ID 变化（如新创建角色）时缺乏自动同步机制。

#### P8: 每轮用户输入的 LLM 调用过多

最低路径：Route(1) + Think(1) = 2 次调用
含委派：Route(1) + Think(1) + SubAgent(1-N) + 回到Think(1) = 3-12+ 次
含综合：再加 1 次

复杂的战斗场景可能产生 **20-30+ 次 LLM 调用**。

#### P9: set_phase 打破读写分离原则

`agents.go:164` 将 `set_phase` 注册为 **MainAgent 可直接调用的写工具**，这打破了项目核心设计原则（"game_engine NEVER performs D&D rule calculations"，MainAgent 应只通过 `delegate_task` 触发写操作）。虽然是 Phase 管理（而非规则运算），但这为后续在 MainAgent 中直接添加更多写操作开了先例，需要明确边界。

#### P10: 上下文压缩器缺少幻觉检测

新代码已实现了 `ContextCompressor`（`llm/context_compressor.go`），包括：
- 两级压缩（Prune 工具输出 + LLM 结构化摘要 + 启发式回退）
- D&D 领域感知（查询类工具激进压缩，动作类保守压缩）
- 异步后台压缩（不阻塞主流程）
- 基于实际 token 使用的 EMA 校准

但缺少 **幻觉循环检测**：当 LLM 连续 3 次执行相同工具并得到相同结果时，压缩器不会介入。

---

## 2. 核心设计原则

基于行业研究和 D&D 游戏的特殊需求，确立以下原则：

### 原则 1: 少即是多（Less Tools Per Call）

> 每个 LLM 调用中暴露的工具数量 ≤ 15（目标 ≤ 10）

**依据**：OpenAI 官方建议 + Semantic Kernel 文档 + 多项实验研究均表明，tool 数量超过 10-20 后选择准确率急剧下降。

### 原则 2: 上下文为王（Context Is King）

> 工具选择应基于当前游戏上下文动态裁剪，而非静态暴露全部工具

**依据**：API-Bank 三层架构与 Semantic Kernel 的 "only import necessary plugins for your scenario" 原则。

### 原则 3: 隐藏复杂性（Hide Complexity from LLM）

> LLM 不应看到的就不能看到。会用错的参数应由系统自动注入

**依据**：Semantic Kernel 最佳实践——"Inject context server-side; the LLM shouldn't track session IDs or auth tokens."

### 原则 4: 合并优于碎片（Consolidate Over Fragment）

> 将细粒度 API 合并为语义完整的复合工具，减少工具数量、降低选择难度

**依据**：Semantic Kernel "Strike the right balance between number of functions and their responsibilities."

### 原则 5: 计划优先于行动（Plan-Then-Act）

> 对复杂操作（尤其是战斗），先生成计划再逐步执行，而非逐步 ReAct

**依据**：GITM（Mehta et al. 2023）证明 plan-first-then-act 在游戏任务中优于纯 ReAct；Tree of Thoughts 在复杂推理中更优。

---

## 3. 改进方案总览

| 优先级 | 方案 | 核心收益 | 复杂度 |
|--------|------|---------|--------|
| **P0** | 动态 Tool 过滤 | 每个 LLM 调用工具数从 36 降至 ≤10 | 中 |
| **P0** | 亚代理工具隔离 + 描述增强 | SubAgent 工具选择准确率提升 | 低 |
| **P0** | 复合工具合并 | 工具总数从 ~97 降至 ~30 | 高 |
| **P1** | 状态引用 + 上下文自动注入 | 参数错误率大幅下降 | 中 |
| **P1** | 游戏状态摘要增强 | 减少冗余工具调用 | 低 |
| **P1** | ReAct 改进（反射检测 + 计划优先） | 减少重复调用和幻觉循环 | 中 |
| **P2** | 路由统一 + 代理覆盖补全 | 解锁 8 个不可达代理 | 低 |
| **P2** | 持久化记忆与反思 | NPC 行为更智能 | 高 |
| **P2** | 验证代理 | 防止级联错误 | 中 |

---

## 4. P0: 动态 Tool 过滤与上下文裁剪

### 4.1 问题

当前 MainAgent 每次 LLM 调用都发送 36 个 Tool Schema（35 readOnly + delegate_task），不考虑游戏阶段或当前上下文。

### 4.2 方案: 基于游戏阶段的动态工具集

在 `ToolRegistry` 中新增 `GetByPhase(phase Phase)` 方法，根据当前游戏阶段返回不同的工具子集：

```go
type Phase string

const (
    PhaseCharacterCreation Phase = "character_creation"
    PhaseExploration       Phase = "exploration"
    PhaseCombat            Phase = "combat"
    PhaseRest              Phase = "rest"
    PhaseNPCInteraction    Phase = "npc_interaction"
    PhaseCrafting          Phase = "crafting"
)

func (r *ToolRegistry) GetByPhase(phase Phase) []Tool {
    baseTools := r.phaseTools["base"] // 始终包含的工具
    phaseTools := r.phaseTools[phase]  // 当前阶段特定工具
    return append(baseTools, phaseTools...)
}
```

### 4.3 各阶段工具分配

#### 基础工具（始终可用）

| 工具 | 用途 |
|------|------|
| `delegate_task` | 委派任务给 SubAgent |
| `get_current_scene` | 当前场景信息 |
| `get_actor` | 获取角色信息 |
| `get_pc` | 获取玩家角色信息 |
| `list_actors` | 列出所有角色 |
| `get_current_combat` | 当前战斗状态 |
| `get_quest` / `list_quests` | 任务信息 |
| `get_npc_attitude` | NPC 态度 |
| **共 8 个** | |

#### 角色创建阶段

| 工具 | 用途 |
|------|------|
| `create_pc` | 创建角色 |
| `list_races` | 列出种族 |
| `get_race` | 种族详情 |
| `list_classes` | 列出职业 |
| `get_class` | 职业详情 |
| `list_backgrounds` | 列出背景 |
| `get_background` | 背景详情 |
| `list_feats_data` | 列出专长 |
| `get_feat_data` | 专长详情 |
| **共 8 + 8(base) = 16 个** | |

#### 探索阶段

| 工具 | 用途 |
|------|------|
| `create_scene` / `update_scene` | 场景管理 |
| `set_current_scene` | 切换场景 |
| `add_scene_connection` | 场景连接 |
| `move_actor_to_scene` | 角色移动 |
| `list_scenes` / `get_scene` | 场景信息 |
| `get_scene_actors` / `get_scene_items` | 场景内容 |
| `interact_with_npc` | NPC 交互 |
| `perform_skill_check` | 技能检定 |
| `perform_ability_check` | 能力检定 |
| **共 ~12 + 8(base) = 20 个** | |

> 注：结合 P0 中的复合工具合并方案后，探索阶段的独立工具数可降至 ≤15

#### 战斗阶段

| 工具 | 用途 |
|------|------|
| `execute_attack` | 执行攻击 |
| `execute_action` | 执行动作 |
| `execute_damage` / `execute_healing` | 伤害/治疗 |
| `next_turn` | 下一回合 |
| `cast_spell` | 施法 |
| `perform_saving_throw` | 豁免检定 |
| `perform_death_save` | 死亡豁免 |
| `concentration_check` | 集中检定 |
| `show_combat_status` | 战斗状态（新增复合工具） |
| `show_available_targets` | 可用目标（新增复合工具） |
| **共 ~10 + 8(base) = 18 个** | |

> 注：结合复合工具合并后可降至 ≤12

### 4.4 SubAgent 动态工具过滤

SubAgent 调用时也应根据任务类型裁剪工具集：

```go
func (r *ToolRegistry) GetByAgentAndPhase(agent string, phase Phase) []Tool {
    agentTools := r.GetByAgent(agent)
    agentPhaseTools := r.agentPhaseTools[agent][phase]
    if agentPhaseTools != nil {
        return agentPhaseTools
    }
    return agentTools // 回退：返回该代理的全部工具
}
```

### 4.5 实现要点

1. 在 `ToolRegistry` 中维护 `phaseTools map[Phase][]string` 映射
2. `ReActLoop.PhaseObserve` 中收集当前 phase，传递给 `MainAgent.Execute()`
3. `MainAgent.Execute()` 使用 `GetByPhase()` 替代 `GetReadOnlySchemas()`
4. 删除系统提示中的内联工具名称列表（消除双重冗余）
5. 添加 phase 自动检测逻辑：如果 `engine.GetPhase()` 不可靠，基于状态推断（如有活跃战斗 → combat）

---

## 5. P0: 亚代理工具隔离与描述增强

### 5.1 SubAgent 工具隔离

当前 SubAgent 调用时暴露其全部注册工具（不区分读写）。改进方案：

**为 SubAgent 引入 readOnly/write 分离**：

```go
type SubAgent interface {
    Agent
    CanHandle(ctx AgentContext, task string) bool
    Priority() int
    Dependencies() []string
    
    // 新增：根据任务类型返回应暴露的工具
    ToolsForTask(task string) []Tool
}
```

`ToolsForTask()` 方法让每个 SubAgent 根据具体任务返回最相关的工具子集，而非全量暴露：

```go
func (a *CombatAgent) ToolsForTask(task string) []Tool {
    switch {
    case strings.Contains(task, "attack"):
        return []Tool{a.tools["execute_attack"], a.tools["show_available_targets"]}
    case strings.Contains(task, "spell"):
        return []Tool{a.tools["cast_spell"], a.tools["concentration_check"]}
    case strings.Contains(task, "death"):
        return []Tool{a.tools["perform_death_save"], a.tools["execute_healing"]}
    default:
        return a.Tools() // 回退到全量
    }
}
```

### 5.2 Tool 描述增强

当前工具描述过于简单，LLM 无法区分相似工具。改进方案——为每个工具描述添加结构化上下文：

**描述模板**：

```
{功能简述}

使用场景: {何时使用此工具}
不使用场景: {何时不使用此工具，应使用哪个替代工具}
参数说明:
  - {param}: {详细说明，包括类型约束和取值范围}
返回值: {返回数据结构简述}
示例:
  输入: {典型参数}
  输出: {典型返回值}
```

**改造前后对比**：

改造前：
```
"execute_attack": "Execute an attack in combat"
```

改造后：
```
"execute_attack": "Execute a melee or ranged attack against a target in active combat.

Use when: You need to make an attack roll during combat (weapon attack, unarmed strike, etc.)
Do NOT use when: Casting a spell attack — use cast_spell instead.

Parameters:
  - target_actor_id (required): ID of the target actor. Use show_available_targets to get valid IDs. Do NOT use character names.
  - attack_type (required): 'melee' or 'ranged'
  - weapon_name (optional): Name of the weapon being used

Returns: Attack result including roll, damage dealt, and target HP status."
```

### 5.3 删除内联工具描述

从 `main_system.md` 中删除 `{{range .ReadOnlyTools}}` 列表。工具信息已经通过 function-calling schema 传递，内联列表浪费 context token 并造成冗余。

将模板变量 `ReadOnlyTools` 替换为对当前可用代理和职责的简短描述：

```markdown
## Available Agents

You can delegate tasks to the following specialized agents:
- **character_agent**: Create and manage characters (PC, NPC, Enemy, Companion)
- **combat_agent**: Handle all combat actions (attacks, spells, damage, healing, death saves)
- **rules_agent**: Perform checks, saves, cast spells, and manage rest periods
- **narrative_agent**: Manage scenes, exploration, travel, and world events
- **inventory_agent**: Manage items, equipment, currency, and attunement
... etc.
```

---

## 6. P0: 复合工具（Composite Tools）合并细粒度 API

### 6.1 问题

当前 dnd-core 引擎的 API 是面向程序员的细粒度接口，不适合直接暴露给 LLM。例如，执行一次攻击需要 LLM 自己按顺序调用 `list_visible_actors` → `execute_attack` → （可能）`concentration_check` → `execute_damage`，这要求 LLM 理解 D&D 5e 的复杂规则流程。

### 6.2 方案: 复合工具

将多个底层 API 合并为语义完整的复合工具。复合工具内部按规则编排底层 API 调用序列，对 LLM 只暴露一个高层接口。

#### 复合工具设计模式

```go
type CompositeTool struct {
    BaseTool
    steps []ToolStep  // 内部执行步骤
}

type ToolStep struct {
    Tool     Tool
    Params   func(ctx ToolContext) map[string]any  // 动态参数映射
    OnResult func(result ToolResult, ctx ToolContext)  // 结果处理
}
```

#### 合并方案

##### 战斗系统合并

| 当前独立工具 | 合并为 | 复合工具行为 |
|-------------|--------|-------------|
| `list_visible_actors` + `execute_attack` + `execute_damage` | `combat_attack` | 自动获取目标列表 → 执行攻击 → 自动计算伤害→ 返回完整攻击结果 |
| `start_combat` + `next_turn` | `combat_start` | 初始化战斗 → 推进到第一回合 → 返回战斗状态 |
| `execute_healing` + `update_actor` | `combat_heal` | 自动计算治疗量 → 更新目标HP → 返回结果 |
| `perform_death_save` + `get_actor` | `combat_death_save` | 执行豁免 → 判断是否稳定/死亡 → 返回完整状态 |
| `get_current_combat` + `get_current_turn` + `list_actors` | `show_combat_status` | 返回战斗完整快照：参战者、HP、当前回合、可用动作 |

**效果**: 战斗工具从 ~12 个降至 ~5 个

##### 角色创建合并

| 当前独立工具 | 合并为 | 复合工具行为 |
|-------------|--------|-------------|
| `list_races` + `get_race` | `query_races` | 返回种族列表含简要描述（无需二次查询） |
| `list_classes` + `get_class` | `query_classes` | 返回职业列表含简要描述 |
| `list_backgrounds` + `get_background` | `query_backgrounds` | 返回背景列表含简要描述 |
| `create_pc` (增强) | `create_character` | 一步创建角色，自动分配初始装备和特性 |

**效果**: 角色工具从 ~12 个降至 ~5 个

##### 数据查询合并

| 当前独立工具 | 合并为 | 复合工具行为 |
|-------------|--------|-------------|
| `list_spells` + `get_spell` | `query_spells` | 支持按等级、职业、学派过滤 |
| `list_weapons` / `list_armors` / `list_magic_items` | `query_equipment` | 支持按类型过滤的统一查询 |
| `list_monsters` + `get_monster` | `query_monsters` | 支持按挑战等级过滤 |
| `list_feats_data` + `get_feat_data` | `query_feats` | 返回列表含简要描述 |

**效果**: 数据查询从 ~15 个降至 ~5 个

##### 场景系统合并

| 当前独立工具 | 合并为 | 复合工具行为 |
|-------------|--------|-------------|
| `create_scene` + `add_scene_connection` | `create_connected_scene` | 创建场景并自动建立连接 |
| `get_scene` + `get_scene_actors` + `get_scene_items` | `show_scene_detail` | 返回场景完整信息 |
| `move_actor_to_scene` + `get_scene` | `move_to_scene` | 移动角色并返回新场景信息 |

**效果**: 场景工具从 ~18 个降至 ~8 个

### 6.3 合并后工具总览

| 代理 | 合并前 | 合并后 | 减少 |
|------|--------|--------|------|
| MainAgent (基础) | 35 readOnly + 1 delegate | ~8 phase-specific + 1 delegate | -78% |
| character_agent | ~9 | ~5 | -44% |
| combat_agent | ~12 | ~5 | -58% |
| rules_agent | ~13 | ~6 | -54% |
| inventory_agent | ~9 | ~4 | -56% |
| narrative_agent | ~18 | ~8 | -56% |
| data_query_agent | ~15 | ~5 | -67% |
| 其他代理 | ~21 | ~10 | -52% |
| **总计** | **~97** | **~46** | **-53%** |

加上动态 Phase 过滤后，MainAgent 实际每次调用暴露的工具数：**8-15 个**（取决于阶段）。

### 6.4 复合工具返回值增强

复合工具应返回结构化的完整信息，避免 LLM 需要再次调用工具获取结果：

```go
type CombatAttackResult struct {
    Success       bool          `json:"success"`
    AttackRoll    int           `json:"attack_roll"`
    TargetAC      int           `json:"target_ac"`
    Hit           bool          `json:"hit"`
    DamageRoll    int           `json:"damage_roll,omitempty"`
    DamageType    string        `json:"damage_type,omitempty"`
    TargetNewHP   int           `json:"target_new_hp,omitempty"`
    TargetMaxHP   int           `json:"target_max_hp,omitempty"`
    TargetStatus  string        `json:"target_status,omitempty"` // alive, dying, dead
    CombatStatus  *CombatSnap   `json:"combat_status,omitempty"` // 完整战斗快照
}
```

---

## 7. P1: 状态引用机制（State IDs）与上下文自动注入

### 7.1 问题

LLM 经常将角色名（如 "Gandalf"）而非 ID（如 "actor_0x3a7f"）传入 `actor_id` 参数，导致工具调用失败。当前 `KnownEntityIDs` 仅支持 `actor_id` 和 `scene_id` 两种类型，不够全面。

### 7.2 方案: 自动上下文注入 + 名称解析

#### 策略 1: 服务端自动注入（LLM 不可见的参数）

对于 `game_id`、`player_id`、当前 `actor_id`（PC 的 ID）、当前 `scene_id` 等 LLM 不应关心的参数，由系统自动注入，不包含在 Tool Schema 中：

```go
type AutoInjectedTool struct {
    BaseTool
    injectFunc func(ctx AgentContext, params map[string]any) map[string]any
}

func (t *AutoInjectedTool) Execute(ctx AgentContext, params map[string]any) (ToolResult, error) {
    // 自动注入参数
    enriched := t.injectFunc(ctx, params)
    return t.BaseTool.Execute(ctx, enriched)
}

// 示例：combat_attack 自动注入 game_id 和 current_actor_id
func injectCombatContext(ctx AgentContext, params map[string]any) map[string]any {
    params["game_id"] = ctx.GameID
    params["attacker_id"] = ctx.KnownEntityIDs["pc_id"]
    return params
}
```

效果：LLM 需要提供的参数从 4-5 个减少到 1-2 个核心参数（如仅 `target_actor_id` 和 `attack_type`）。

#### 策略 2: 名称自动解析

当 LLM 传入名称而非 ID 时，系统自动解析：

```go
type NameResolvingTool struct {
    BaseTool
    resolveParams map[string]string // param_name -> entity_type
}

func (t *NameResolvingTool) Execute(ctx AgentContext, params map[string]any) (ToolResult, error) {
    for paramName, entityType := range t.resolveParams {
        if val, ok := params[paramName]; ok {
            if !strings.HasPrefix(val.(string), "actor_") && !strings.HasPrefix(val.(string), "scene_") {
                // 这是一个名称而非 ID，尝试解析
                resolved, err := resolveName(ctx, val.(string), entityType)
                if err == nil {
                    params[paramName] = resolved
                }
            }
        }
    }
    return t.BaseTool.Execute(ctx, params)
}
```

#### 策略 3: 扩展 KnownEntityIDs

将 `KnownEntityIDs` 扩展为完整的实体注册表：

```go
type EntityRegistry struct {
    Entities map[string]EntityRef  // id -> ref
    ByName    map[string]string    // name -> id
}

type EntityRef struct {
    ID       string
    Name     string
    Type     string  // "actor", "scene", "quest", "item", "spell"
    SubType  string  // "pc", "npc", "enemy", "companion"
}
```

每次工具调用返回新实体时，自动注册到 `EntityRegistry`。后续调用中，LLM 可以使用名称或 ID，系统都正确解析。

---

## 8. P1: 游戏状态摘要增强

### 8.1 问题

当前 `CollectSummary()` 仅返回极简信息（游戏名、场景名、HP、AC），LLM 不得不频繁调用工具获取详细状态。

### 8.2 方案: 富状态快照

```go
type RichGameSummary struct {
    // 基础信息
    GameID   string `json:"game_id"`
    Phase    string `json:"phase"`
    Turn     int    `json:"turn,omitempty"`

    // 场景信息
    CurrentScene SceneSnapshot `json:"current_scene"`

    // 玩家完整状态
    Player PlayerSnapshot `json:"player"`

    // 战斗状态（如有）
    Combat *CombatSnapshot `json:"combat,omitempty"`

    // 活跃任务（标题+目标）
    Quests []QuestSummary `json:"quests,omitempty"`

    // 场景内可见实体（ID+名称，供LLM引用）
    VisibleEntities []EntityRef `json:"visible_entities"`
}

type PlayerSnapshot struct {
    ID            string            `json:"id"`
    Name          string            `json:"name"`
    Race          string            `json:"race"`
    Class         string            `json:"class"`
    Level         int               `json:"level"`
    HP            int               `json:"hp"`
    MaxHP         int               `json:"max_hp"`
    AC            int               `json:"ac"`
    AbilityScores map[string]int   `json:"ability_scores"`
    Conditions    []string          `json:"conditions,omitempty"`
    SpellSlots    map[int]int       `json:"spell_slots,omitempty"`
    KeyInventory  []string          `json:"key_inventory,omitempty"`  // 仅列出关键物品
    EquippedWeapon string           `json:"equipped_weapon,omitempty"`
    EquippedArmor  string           `json:"equipped_armor,omitempty"`
}
```

### 8.3 关键改进

1. **`VisibleEntities`**: 列出当前场景所有可见实体（ID + 名称），LLM 可直接引用，无需调用 `list_actors`
2. **`SpellSlots`**: 显示剩余法术位，避免 LLM 每次施法前都要查询
3. **`Conditions`**: 显示状态效果，避免 LLM 忘记处理中毒/擒抱等
4. **`KeyInventory`**: 仅列出关键物品（如武器、护甲、特殊物品），不列出全部物品栏
5. **选择性丰富**：战斗时包含 `CombatSnapshot`（参战者、HP、回合），非战斗时省略

### 8.4 Token 预算控制

`RichGameSummary` 应支持层级输出：

```go
func (s *RichGameSummary) FormatForLLM(tokenBudget int) string {
    if tokenBudget > 2000 {
        return s.formatFull()      // 完整版本
    } else if tokenBudget > 500 {
        return s.formatCompact()   // 紧凑版本
    }
    return s.formatMinimal()       // 最低版本，等同当前
}
```

根据剩余 context window 动态调整状态摘要的详细程度。

---

## 9. P1: ReAct 循环改进——反射检测与计划优先

### 9.1 问题

当前 ReAct 循环存在以下问题：
- 最大迭代次数 10，但无幻觉检测机制——重复相同动作得到相同结果时会无限循环
- 逐步 ReAct 对战斗等复杂场景效率低，LLM 需要多轮对话才能完成一次攻击
- SubAgent 的 mini-ReAct 循环最多 10 次迭代，可能产生大量 LLM 调用

### 9.2 幻觉循环检测（Reflexion）

在 ReAct 循环中添加重复动作检测：

```go
type ReActLoop struct {
    // ...existing fields...
    
    actionHistory []ActionRecord  // 新增：动作历史
}

type ActionRecord struct {
    ToolName string
    Params   map[string]any
    Result   string
}

// 检测是否陷入循环
func (l *ReActLoop) isHallucinationLoop() bool {
    if len(l.actionHistory) < 3 {
        return false
    }
    
    recent := l.actionHistory[len(l.actionHistory)-3:]
    
    // 连续3次相同动作+相同结果 → 幻觉循环
    sameTool := recent[0].ToolName == recent[1].ToolName && recent[1].ToolName == recent[2].ToolName
    sameParams := reflect.DeepEqual(recent[0].Params, recent[1].Params) && reflect.DeepEqual(recent[1].Params, recent[2].Params)
    sameResult := recent[0].Result == recent[1].Result && recent[1].Result == recent[2].Result
    
    return sameTool && sameParams && sameResult
}
```

当检测到循环时：
1. 向 LLM 发送警告："你已连续3次执行相同操作并得到相同结果。请尝试不同的方法。"
2. 如果继续循环，强制终止当前 ReAct 迭代，返回已有结果

### 9.3 计划优先模式（Plan-Then-Act）

对战斗等复杂场景，引入两阶段执行：

#### 阶段 1: 生成战术计划

```markdown
## Combat Plan Request

You are in combat. Analyze the situation and create a plan for the current round.
Available actions:
- combat_attack: Attack a target
- cast_spell: Cast a spell
- combat_heal: Heal an ally
- show_combat_status: View battle status

Respond with a JSON plan:
{
  "actions": [
    {"tool": "combat_attack", "params": {"target": "Goblin 1", "type": "melee"}, "reason": "Lowest HP enemy"},
    ...
  ],
  "contingency": "If attack misses, use Dodge action next turn"
}
```

#### 阶段 2: 执行计划

系统按计划顺序执行各动作，每个动作的结果自动注入到下一个动作的上下文中。LLM 仅在计划遭遇异常时重新介入。

```go
func (l *ReActLoop) executePlan(plan CombatPlan, ctx AgentContext) ([]ToolResult, error) {
    results := []ToolResult{}
    for i, action := range plan.Actions {
        result, err := l.registry.Execute(action.Tool, action.Params, ctx)
        if err != nil {
            // 计划失败，回到 LLM 重新决策
            return results, fmt.Errorf("plan step %d failed: %w", i, err)
        }
        results = append(results, result)
        
        // 将结果注入上下文，供下一步使用
        ctx.AgentResults[action.Tool] = result
    }
    return results, nil
}
```

**效果**：战斗场景从 5-10 次 LLM 调用降至 1-2 次（计划生成 + 异常处理）。

### 9.4 SubAgent 迭代上限降低

将 SubAgent mini-ReAct 最大迭代从 10 降低到 3-5，并增加单次迭代结果检查：

```go
const subAgentMaxIterations = 5  // 从 10 降低

// 在每次迭代后检查任务是否已完成
func (l *ReActLoop) isSubAgentTaskComplete(results []ToolResult) bool {
    for _, r := range results {
        if r.Success && r.Metadata["task_complete"] == "true" {
            return true
        }
    }
    return false
}
```

复合工具在返回结果时设置 `task_complete` 标记，帮助循环提前终止。

---

## 10. P2: 路由层统一与代理覆盖补全

### 10.1 问题

`delegate_task` 和 `RouterAgent` 仅暴露 3 个代理，8 个代理完全不可达。

### 10.2 方案: 统一路由决策

#### 补全 `delegate_task` 的 `agent_name` 枚举

```go
// delegate_task_tool.go
var AgentNameEnum = []string{
    "character_agent", "combat_agent", "rules_agent",
    "narrative_agent", "inventory_agent", "npc_agent",
    "memory_agent", "movement_agent", "mount_agent",
    "crafting_agent", "data_query_agent",
}
```

#### 合并路由与委派

当前存在两套路由机制（`RouterAgent.Route()` 和 `delegate_task`），功能重叠。改进方案：

**方案 A（推荐）: 保留 delegate_task，删除 RouterAgent**

MainAgent 通过 `delegate_task` 直接委派给 SubAgent，无需额外的路由 LLM 调用。这消除了 Route 阶段的 LLM 调用，节省一次 API 调用。

```
改进前: User → ReAct(Observe) → ReAct(Route) → ReAct(Think) → ...
改进后: User → ReAct(Observe) → ReAct(Think) → ...
```

**方案 B: 保留两个阶段，但共享路由知识**

如果 Route 阶段被认为有价值（如防止 MainAgent 委派到错误代理），则：
- 让 Router 和 delegate_task 使用相同的代理列表和描述
- 将路由知识提取为共享的 `AgentCatalog`
- Router 在 PhaseRoute 进行轻量级分类后，MainAgent 在 Think 阶段仍可推翻路由建议

```go
type AgentCatalog struct {
    agents map[string]AgentInfo
}

type AgentInfo struct {
    Name        string
    Description string
    Capabilities []string
    Examples    []string // 示例任务
}

var DefaultCatalog = AgentCatalog{
    agents: map[string]AgentInfo{
        "character_agent": {
            Name:        "character_agent",
            Description: "Create and manage player characters, NPCs, enemies, and companions",
            Capabilities: []string{"create_pc", "create_npc", "create_enemy", "update_actor", "add_experience"},
            Examples:    []string{"Create a new character", "Level up a character", "Update an NPC's description"},
        },
        // ... 其他10个代理
    },
}
```

### 10.3 采用方案 A 的理由

- 减少 1 次 LLM 调用（每次用户输入节省约 30% token）
- MainAgent 已经有足够的上下文来决定委派
- 复合工具减少后，MainAgent 的决策空间更小，委派更准确
- Router 的路由建议可能被 MainAgent 推翻，增加了不确定性而非减少

---

## 11. P2: 持久化记忆与反思机制

### 11.1 灵感来源

来自 Park et al. (2023) Generative Agents 论文：
- **记忆流 (Memory Stream)**: 自然语言观察序列
- **检 索**: 基于相关性、时效性、重要性检索记忆
- **反思 (Reflection)**: 从低层观察中综合出高层洞察

### 11.2 应用于 D&D

```
观察 → "玩家在洞穴入口遇到了哥布林巡逻队"
                ↓
记忆流 → [时间戳] 在洞穴入口遇到哥布林. 哥布林数量3. 玩家选择了战斗.
                ↓
反思   → "玩家倾向于用战斗解决问题，但当前HP较低(12/30)，应建议谨慎战术"
```

### 11.3 实现架构

```go
type Memory struct {
    store      MemoryStore
    importance  ImportanceScorer  // 重要性评分
    retrieval   MemoryRetriever    // 基于相关性+时效性+重要性检索
}

type MemoryEntry struct {
    ID         string
    Content    string
    Timestamp  time.Time
    Importance float64   // 1-10 分
    Tags       []string  // "combat", "npc_interaction", "quest"
}

type Reflection struct {
    ID         string
    Content    string    // 高层洞察
    DerivedFrom []string // 关联的记忆 ID
    Timestamp  time.Time
}
```

### 11.4 记忆注入系统提示

反思结果作为系统提示的一部分注入：

```markdown
## Relevant Memories

- The player tends to rush into combat without scouting (importance: 8)
- Last session, the player negotiated with the dragon instead of fighting (importance: 7)
- The player's character has low HP and no healing potions (importance: 9)

## Strategic Reflection

Based on recent patterns, the player should be offered tactical retreat options when HP is below 50%. 
The party lacks a dedicated healer, so healing resources should be managed carefully.
```

### 11.5 分阶段实施

- **Phase 2.5**: 实现基础 `MemoryStore`（SQLite/in-memory）+ 简单重要性评分
- **Phase 3**: 实现 `Reflection` 生成（使用 LLM 综合观察 → 洞察）+ 检索增强
- **Phase 3.5**: NPC 记忆系统（每个 NPC 独立记忆流）

---

## 12. P2: 验证代理（Verification Agent）

### 12.1 问题

在 D&D 这种规则密集型领域，LLM 可能在不理解规则的情况下执行错误操作（如错误计算伤害类型、错误应用状态效果）。同一个 LLM 用来"自我验证"是不可靠的（ChemCrow 论文的教训）。

### 12.2 方案: 规则验证代理

引入轻量级验证代理，在关键操作前进行规则校验：

```go
type VerificationAgent struct {
    llm       llm.LLMClient
    ruleBook  map[string]RuleCheck  // 规则检查函数
}

type RuleCheck func(params map[string]any, state GameSummary) *RuleViolation

// 示例：验证施法条件
func spellCastingCheck(params map[string]any, state GameSummary) *RuleViolation {
    spellLevel := params["level"].(int)
    playerSlots := state.Player.SpellSlots[spellLevel]
    if playerSlots <= 0 {
        return &RuleViolation{
            Rule:    "spell_slots_exhausted",
            Message: fmt.Sprintf("%s has no level %d spell slots remaining", state.Player.Name, spellLevel),
        }
    }
    return nil
}
```

### 12.2 验证触发机制

并非所有操作都需要验证。验证仅在以下情况触发：

1. **不可逆操作**: 角色死亡、物品消耗、任务完成
2. **复合工具执行前**: 确保 Plan-Then-Act 的计划符合规则
3. **LLM 信心低时**: 检测到 LLM 选择工具时的概率分布较平

### 12.3 轻量级实现

验证代理不需要 LLM 调用——它使用 dnd-core 引擎的规则逻辑进行确定性检查：

```go
func (v *VerificationAgent) Verify(tool string, params map[string]any, state GameSummary) error {
    checks, ok := v.ruleBook[tool]
    if !ok {
        return nil // 无验证规则，放行
    }
    for _, check := range checks {
        if violation := check(params, state); violation != nil {
            return fmt.Errorf("rule violation [%s]: %s", violation.Rule, violation.Message)
        }
    }
    return nil
}
```

这确保了关键操作符合 D&D 5e 规则，而不依赖 LLM 自我检查。

---

## 13. 已有的改进与差距分析

> 本节对照最新代码（`refactor-sub-agent` 分支），分析哪些改进已经实现、哪些还未实现、以及哪些需要调整。

### 13.1 已实现 ✅

| 改进 | 位置 | 说明 |
|------|------|------|
| **上下文压缩器** | `llm/context_compressor.go` | 完整实现：两级压缩、异步后台、D&D 领域感知、EMA 校准 |
| **Phase 管理** | `tool/phase_tools.go` | `set_phase`(写) + `get_phase`(读) 工具已注册 |
| **Phase 自动推进** | `react_loop.go:276-287` | 在 Observe 阶段自动从 character_creation → exploration |
| **游戏阶段状态** | `game_summary/summary.go:138-142` | `GameSummary.Phase` 字段已就位 |
| **Token 校准** | `react_loop.go:451-455` | 压缩器利用 LLM 实际返回的 PromptTokens 进行 EMA 校准 |
| **SubAgent 隔离上下文** | `react_loop.go:845-922` | `createSubSession` 从父会话提取最近 20 条消息作为上下文 |

### 13.2 未实现但有基础设施 ❌→🔧

| 改进 | 差距 | 需要的调整 |
|------|------|-----------|
| **Phase 感知的 Tool 过滤** | Phase 枚举和状态已就位，但 `ToolRegistry` 没有 `GetByPhase()` 方法 | 新增 `phaseTools` 映射 + `GetByPhase()` 方法，在 `MainAgent.Execute()` 和 SubAgent 调用时使用 |
| **delegate_task 扩展到全部代理** | 仅 3 个代理可委派，8 个代理不可达 | 扩展 `delegate_task_tool.go:24` 的 enum + `router_agent.go:128` 的 enum + `main_system.md` 的代理列表 |
| **游戏状态摘要增强** | `GameSummary` 仅有 Name/HP/AC/Scene/Quest | 扩展 `ActorSummary`、`CombatSummary`、新增 `VisibleEntities` 等 |

### 13.3 完全未实现 ❌

| 改进 | 说明 |
|------|------|
| 复合工具（Composite Tools） | 无任何基础设施，需要新的 Tool 包装器 |
| 工具描述增强（Use/Do NOT use） | 当前工具描述太简单 |
| 删除内联工具列表 | `main_system.md:30-32` 仍用 `{{range .ReadOnlyTools}}` 列出 |
| 名称自动解析 | `KnownEntityIDs` 仍只支持 `actor_id` 和 `scene_id` |
| 服务端自动注入参数 | `game_id` 仍需 LLM 手动传入 |
| SubAgent 工具按任务过滤 | SubAgent 仍全量暴露所有注册工具 |
| SubAgent 迭代上限降低 | 仍为 10（`react_loop.go:740`） |
| 幻觉循环检测 | ReAct 循环无重复动作检测 |
| 计划优先模式（Plan-Then-Act） | 无任何基础 |
| 验证代理 | 无 |
| 持久化记忆 | 无 |

### 13.4 需要调整的设计

#### set_phase 的读写分离问题

`agents.go:164` 将 `set_phase` 注册为 MainAgent 可直接调用的写工具：

```go
registry.Register(tool.NewSetPhaseTool(engine), []string{agent.MainAgentName, agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.SubAgentNameRules}, "phase")
```

这与项目核心原则 "MainAgent 只能通过 delegate_task 触发写操作" 矛盾。合理原因是：Phase 管理属于**游戏流程控制**而非 D&D 规则运算，可以由 DM（MainAgent）直接决定。

**建议**：在 AGENTS.md 和架构文档中明确分类两种写操作：
1. **规则写操作**（伤害、治疗、检定等）→ 必须通过 `delegate_task`
2. **流程写操作**（Phase 推进等）→ MainAgent 可直接调用

`MainAgent.separateCallsByAccess()` 中的写操作拦截逻辑也需要相应调整，区分 `readOnly` / `write_rule` / `write_flow` 三类。

#### 压缩器与新方案的协同

现有 `ContextCompressor` 的 Prune 阶段按工具名前缀（`get_`、`list_`、`query_`）识别查询类工具进行激进压缩（`context_compressor.go:438-448`）。当实施复合工具后，工具命名将变化（如 `combat_attack` 替代 `execute_attack`），需要确保压缩器的前缀列表同步更新。

建议将 `isQueryTool()` 改为基于 `Tool.ReadOnly()` 标记而非命名前缀，使其与复合工具自动兼容：

```go
// 改进前：基于命名前缀
func isQueryTool(name string) bool {
    for _, prefix := range queryToolPrefixes {
        if strings.HasPrefix(name, prefix) { return true }
    }
    return false
}

// 改进后：基于 Tool 注册表的 ReadOnly 标记
func (c *ContextCompressor) isQueryTool(name string) bool {
    if c.registry != nil {
        if t, ok := c.registry.Get(name); ok {
            return t.ReadOnly()
        }
    }
    // 回退到前缀匹配
    for _, prefix := range queryToolPrefixes {
        if strings.HasPrefix(name, prefix) { return true }
    }
    return false
}
```

#### 压缩器需要 ToolRegistry 引用

当前 `ContextCompressor` 不持有 `ToolRegistry` 引用（`context_compressor.go:14-39`）。为支持基于 `ReadOnly()` 的查询判断，需要注入 registry 引用。可以在 `NewReActLoop()` 中将 `tools` 传入压缩机：

```go
func NewReActLoop(...) *ReActLoop {
    // ...
    compressor := llm.DefaultContextCompressor(llmClient)
    compressor.SetToolRegistry(tools) // 新增方法
    // ...
}
```

---

## 14. 实施路线图

### Phase 2.1 (立即实施，1-2 周)

| 改进 | 工作量 | 影响 | 状态 |
|------|--------|------|------|
| 补全 `delegate_task` 枚举（暴露全部 11 个代理） | 小 | 解锁 8 个不可达代理 | ✅ 已完成 |
| 删除系统提示中的内联工具列表 | 小 | 消除 token 冗余 | ✅ 已完成 |
| 增强工具描述（15 个最常用工具） | 小 | 提升工具选择准确率 | ✅ 已完成 |
| 幻觉循环检测（3次重复检测） | 小 | 减少无限循环 | ✅ 已完成 |
| SubAgent 最大迭代降至 5 | 极小 | 减少 LLM 调用 | ✅ 已完成 |
| 最大委托次数提升至 15 | 极小 | 适配 11 代理架构 | ✅ 已完成 |
| 明确 `set_phase` 读写分类边界 | 极小 | 防止设计原则被侵蚀 | ✅ 已完成 |
| 压缩器 `isQueryTool` 改用 `ReadOnly()` 标记 | 小 | 与复合工具兼容 | ✅ 已完成 |
| 路由器 `buildRouterTools` 动态化 | 小 | 新增代理时自动更新 | ✅ 已完成 |

### Phase 2.2 (2-4 周)

| 改进 | 工作量 | 影响 | 状态 |
|------|--------|------|------|
| 基于 Phase 的动态工具过滤 | 中 | MainAgent 工具数降至 8-25 | ✅ 已完成 |
| 游戏状态摘要增强（RichGameSummary） | 中 | 减少冗余工具调用 | ✅ 已完成 |
| 自动上下文注入 + 名称解析机制 | 中 | 消除 name-vs-ID 错误 | ✅ 已完成 |
| 删除 RouterAgent，保留 delegate_task | 小 | 减少 1 次 LLM 调用 | ✅ 已完成 |

### Phase 2.3 (4-8 周)

| 改进 | 工作量 | 影响 | 状态 |
|------|--------|------|------|
| 复合工具实现（战斗系统优先） | 高 | 战斗工具从 12 降至 5 | ❌ 待做 |
| 复合工具实现（角色/场景/查询） | 高 | 总工具数从 ~97 降至 ~46 | ❌ 待做 |
| SubAgent ToolsForTask() 过滤 | 中 | SubAgent 工具数降至 5-8 | ❌ 待做 |
| 战斗 Plan-Then-Act 模式 | 中 | 战斗 LLM 调用减少 50-80% | ❌ 待做 |

### Phase 3.x (与现有规划并行)

| 改进 | 工作量 | 影响 | 状态 |
|------|--------|------|------|
| 持久化记忆系统 | 高 | 提升叙事连贯性 | ❌ 待做 |
| 验证代理 | 中 | 减少规则错误 | ❌ 待做 |
| NPC 记忆系统 | 高 | NPC 行为更自然 | ❌ 待做 |
| 反思机制 | 高 | 长期策略能力 | ❌ 待做 |

### 已完成 ✅

| 改进 | 位置 |
|------|------|
| 上下文压缩器（两级压缩 + 异步 + D&D 感知 + EMA 校准） | `llm/context_compressor.go` |
| Phase 管理工具（set_phase + get_phase） | `tool/phase_tools.go` |
| Phase 自动推进（character_creation → exploration） | `react_loop.go:276-287` |
| 游戏阶段状态字段 | `game_summary/summary.go` |
| delegate_task 枚举扩展到全部 11 个代理 | `tool/delegate_task_tool.go` |
| 路由器 `route_decision` 动态生成代理列表 | `agent/router_agent.go` |
| MainAgent 系统提示删除内联工具列表 + 扩展代理到 11 个 | `prompt/main_system.md` |
| 路由器系统提示扩展代理职责到 11 个 | `prompt/router_system.md` |
| 幻觉循环检测（连续 3 次相同操作警告） | `react_loop.go` |
| SubAgent 最大迭代 10→5 | `react_loop.go` |
| 最大委托次数 10→15 | `react_loop.go` |
| 压缩器 `isQueryTool` 改用 `ReadOnly()` 接口 | `llm/context_compressor.go` |
| `ToolRegistry.IsToolReadOnly()` 实现 `ToolReadOnlyChecker` 接口 | `tool/registry.go` |
| `set_phase` 注册注释标注 write_flow 分类 | `agents.go` |
| MainAgent 删除 `ReadOnlyTools` 模板变量构建 | `agent/main_agent.go` |
| 15 个常用工具描述增强（Use when / Do NOT use when） | `tool/*.go` |
| 基于 Phase 的动态工具过滤（4 阶段裁剪） | `tool/registry.go`, `agent/main_agent.go` |
| 游戏状态摘要增强（属性值/种族/背景/等级/熟练加值/灵感/状态效果） | `game_summary/summary.go`, `formatter.go` |
| 名称解析机制（actor_id/scene_id 名称→ID 自动转换） | `react_loop.go` |
| RouterAgent 默认禁用（节省 1 次 LLM 调用） | `engine.go` |

---

## 15. 参考资料

1. **OpenAI Function Calling Best Practices** — OpenAI 官方文档，建议工具数量 ≤10-20
2. **Semantic Kernel Plugin Design Guidelines** — Microsoft，提出 Plugin 分组、上下文注入、工具合并等原则
3. **API-Bank: Benchmarking Tool-Augmented LLMs** (Li et al., 2023) — 三层 API 调用评估框架
4. **ReAct: Synergizing Reasoning and Acting in Language Models** (Yao et al., 2023) — ReAct 模式原始论文
5. **Reflexion: Language Agents with Verbal Reinforcement Learning** (Shinn & Labash, 2023) — 反思机制，幻觉检测
6. **Generative Agents: Interactive Simulacra of Human Behavior** (Park et al., 2023) — 记忆流、反思、NPC 行为模型
7. **HuggingGPT: Solving AI Tasks with LLMs** (Shen et al., 2023) — 层次化代理路由模式
8. **LangGraph Multi-Agent Patterns** — Supervisor/Swarm/Network 模式
9. **ChemCrow: Augmenting LLMs with Chemistry Tools** (Bran et al., 2023) — 领域专家工具描述的重要性
10. **GITM: Reasoning with Language Model is Planning with World Model** (Mehta et al., 2023) — Plan-First-Then-Act 模式
11. **LLM+P: Empower Large Language Models with Optimal Planning Proficiency** (Liu et al., 2023) — 外部规划器集成

---

## 附录 A: 当前架构 vs 改进架构

### 当前架构

```
Player Input
    ↓
ReAct(Observe) → CollectSummary() [5字段极简状态 + Phase]
    │               └─ Auto-advance: character_creation → exploration ✅
    │               └─ maybeCompressHistory() [异步压缩] ✅
    │               └─ populateKnownEntityIDs [仅 actor_id, scene_id]
    ↓
ReAct(Route) → RouterAgent.Route() [1 LLM call, 仅3个代理 ⚠️]
    ↓
ReAct(Think) → MainAgent.Execute() [36 tool schemas ⚠️]
    │               └─ GetReadOnlySchemas() + delegate_task [全量暴露]
    │               └─ system prompt 含内联工具列表 [双重冗余 ⚠️]
    ↓
ReAct(Act) → delegate_task [仅3个代理 ⚠️] / readOnly tools
    │               └─ set_phase MainAgent可直接调用 [✅ 但打破读写分离]
    │               └─ token校准 回写到压缩机 ✅
    ↓
SubAgent Mini-ReAct [10 iterations ⚠️, 全量工具] → Engine APIs [97个细粒度API]
    │
    ↓
ReAct(Synthesize/Respond)
    └─ Context Compressor: 两级压缩 ✅, 领域感知 ✅, 异步 ✅, 幻觉检测 ❌
```

### 改进架构

```
Player Input
    ↓
ReAct(Observe) → RichGameSummary() [15+字段完整状态 + 可见实体 + Phase]
    │               └─ maybeCompressHistory() ✅ [基于 ReadOnly() 改进]
    │               └─ populateKnownEntityIDs [扩展: actor_id, scene_id, item_id, spell_id...]
    │               └─ NameResolver [name → ID 自动解析]
    ↓
ReAct(Think) → MainAgent.Execute() [≤15 phase-filtered tools]
    │               └─ 删除内联工具列表，仅靠 function-calling schema
    │               └─ 游戏状态含 VisibleEntities，减少查询工具调用
    │               └─ set_phase 分类为 write_flow，允许 MainAgent 直接调用
    ↓
ReAct(Act) → delegate_task [11个代理 ✅] / readOnly tools / write_flow tools
    │               └─ game_id, pc_id 自动注入 [服务端隐藏参数]
    ↓
SubAgent Execute [≤5 iterations, task-filtered tools, ≤8 per call]
    ├→ Composite Tools → [多个底层 Engine APIs 自动编排]
    ├→ Auto-inject game_id/pc_id ✅ (需扩展)
    ├→ Name Resolution (name → ID)
    └→ VerificationAgent.Verify() [关键操作预检]
    ↓
Hallucination Detection [3次重复检测] ← 新增，集成到 ContextCompressor
    ↓
ReAct(Respond)
```

### 关键改进指标

| 指标 | 当前 | 改进后 | 改善 |
|------|------|--------|------|
| MainAgent 每次 Tool 数 | 36 | 8-15 | -58% ~ -78% |
| SubAgent 每次 Tool 数 | 9-15 | 5-8 | -33% ~ -67% |
| 总 Tool 数 | ~97 | ~46 | -53% |
| 每轮 LLM 调用数(最少) | 2 | 1 | -50% |
| 每轮 LLM 调用数(战斗) | 5-10 | 1-2 | -70% ~ -90% |
| 可达代理数 | 3/11 | 11/11 | +267% |
| 状态摘要完整度 | 5个字段 | 15+个字段 | +200% |
| 名称/ID错误率 | 高 | 近零 | 大幅改善 |