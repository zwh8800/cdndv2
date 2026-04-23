# 独立战斗Agent系统 - 实现计划

## Context

当前 cdndv2 的战斗系统作为 SubAgent 运行在 MainAgent 的 `delegate_task` 委派下，每次只能处理单个战斗动作（最多5次迭代的mini-ReAct循环）。这导致：

1. 战斗Agent无法维持跨回合的战斗上下文，每次委派都是独立的
2. 战斗逻辑与叙事耦合，MainAgent需要理解战斗规则来协调回合
3. 敌人回合无法自主决策，需要MainAgent手动协调
4. LLM需要从10+个细粒度工具中选择，容易出错

**目标**：创建一个独立的战斗Agent，在战斗阶段完全接管控制权，使用独立对话历史，管理完整的多回合战斗流程，战斗结束后生成摘要返还控制权。同时在 dnd-core 中实现组合型API，将10+个战斗工具收敛为3个强力接口。

## 架构总览

```
ProcessInput(input)
  │
  ├─ phase != combat → 正常 ReActLoop（MainAgent）
  │
  └─ phase == combat → CombatSession.ProcessInput(input)
                          │
                          ├─ 玩家回合：DM描述 → 等待输入 → 解析执行
                          ├─ 敌人回合：DM描述 → EnemyAIAgent生成意图 → 解析执行
                          └─ 战斗结束：生成summary → 注入主历史 → 还原控制权
```

---

## 实现分6个步骤

### Step 1: dnd-core — 可用动作计算引擎

**目标**：在 dnd-core 中实现动态可用动作列表计算，这是整个战斗系统的基础。

#### 新文件：`../dnd-core/pkg/engine/available_actions.go`

定义核心类型和计算逻辑：

```go
// AvailableActionsResult 当前角色可用动作的完整列表
type AvailableActionsResult struct {
    ActorID    model.ID           `json:"actor_id"`
    ActorName  string             `json:"actor_name"`
    ActorType  string             `json:"actor_type"` // "pc", "enemy", "npc", "companion"
    Movement   MovementOption     `json:"movement"`
    Actions    []AvailableAction  `json:"actions"`
    BonusActions []AvailableAction `json:"bonus_actions"`
    Reactions  []AvailableAction  `json:"reactions"`
    Conditions []string           `json:"conditions"` // 当前生效的状态效果
}

// AvailableAction 单个可用动作
type AvailableAction struct {
    ID             string         `json:"id"`              // 唯一标识，如 "attack_longsword", "cast_fireball"
    Category       string         `json:"category"`        // "attack", "spell", "standard_action", "class_feature", "item_use"
    Name           string         `json:"name"`            // 显示名称
    Description    string         `json:"description"`     // 规则描述
    CostType       string         `json:"cost_type"`       // "action", "bonus_action", "reaction", "free_action"
    RequiresTarget bool           `json:"requires_target"`
    TargetType     string         `json:"target_type"`     // "single", "area", "self", "none"
    ValidTargetIDs []model.ID     `json:"valid_target_ids,omitempty"`
    Range          string         `json:"range"`           // "5尺触及", "120尺"
    ResourceCost   string         `json:"resource_cost,omitempty"` // "消耗3环法术位"
    DamagePreview  string         `json:"damage_preview,omitempty"` // "1d8+3 挥砍"
    Metadata       map[string]any `json:"metadata"`        // 路由信息：_route, weapon_id, spell_id 等
}

// MovementOption 移动选项
type MovementOption struct {
    Available     bool `json:"available"`
    RemainingFeet int  `json:"remaining_feet"`
}
```

**计算算法**（私有方法 `computeAvailableActions`）分层执行：

1. **状态检查层**：若角色有 Stunned/Paralyzed/Unconscious/Incapacitated（`CantTakeActions`）→ 返回空
2. **资源检查层**：读取 `TurnState` 确定 Action/BonusAction/Reaction 是否已用
3. **候选收集层**：
   - Action可用时：基础D&D动作（Attack/Dash/Dodge/Disengage/Help/Hide/Ready/Search）+ 施法（action时间法术）+ 职业特性动作
   - BonusAction可用时：副手攻击（双持轻武器）+ bonus action法术 + 职业特性附赠动作
   - 自由动作：Action Surge 等（来自 `FeatureHook.GetAvailableActions()` 中 `IsFreeAction==true`）
4. **状态过滤层**：使用已有的 `model.GetConditionEffect()` 过滤 —— Restrained→禁移动, Silenced→禁言语法术, Grappled→速度0
5. **法术过滤层**：检查已准备法术的施法时间、法术位可用性、专注冲突、成分需求
6. **目标计算层**：根据武器射程/法术范围计算合法目标列表

公开API：`func (e *Engine) GetAvailableActions(ctx, req GetAvailableActionsRequest) (*AvailableActionsResult, error)`

#### 修改文件：`../dnd-core/pkg/model/combat.go`

在 `TurnState` 中新增多重攻击跟踪字段：

```go
type TurnState struct {
    // ... 现有字段保持不变 ...
    AttacksRemaining int  `json:"attacks_remaining"` // 本回合剩余攻击次数（Extra Attack）
    AttacksTotal     int  `json:"attacks_total"`     // 本回合总攻击次数
}
```

---

### Step 2: dnd-core — 组合型战斗API

**目标**：实现3个组合API，包装现有低级API。

#### 新文件：`../dnd-core/pkg/engine/combat_composite.go`

**API 1: `SetupCombat`** — 组合型战斗初始化

```go
type SetupCombatRequest struct {
    GameID         model.ID            `json:"game_id"`
    SceneID        model.ID            `json:"scene_id,omitempty"` // 可选，空则自动创建
    ParticipantIDs []model.ID          `json:"participant_ids"`
    IsSurprise     bool                `json:"is_surprise"`
    StealthySide   []model.ID          `json:"stealthy_side,omitempty"`
    Observers      []model.ID          `json:"observers,omitempty"`
}

type SetupCombatResult struct {
    Combat    *CombatInfo            `json:"combat"`
    FirstTurn *EnhancedTurnInfo      `json:"first_turn"` // 第一个行动者的完整回合信息
}
```

内部流程：调用已有 `StartCombat`/`StartCombatWithSurprise` 内部逻辑 → 计算第一个行动者的可用动作 → 返回组合结果。

**API 2: 增强 `NextTurn`** — 回合推进 + 完整态势

```go
// EnhancedTurnInfo 增强版回合信息
type EnhancedTurnInfo struct {
    ActorID          model.ID              `json:"actor_id"`
    ActorName        string                `json:"actor_name"`
    ActorType        string                `json:"actor_type"`
    ActorHP          int                   `json:"actor_hp"`
    ActorMaxHP       int                   `json:"actor_max_hp"`
    ActorAC          int                   `json:"actor_ac"`
    ActorConditions  []string              `json:"actor_conditions"`
    Round            int                   `json:"round"`
    AvailableActions *AvailableActionsResult `json:"available_actions"`
    Participants     []CombatantStatus     `json:"participants"`
    CombatEnd        *CombatEndState       `json:"combat_end,omitempty"`
}

// CombatantStatus 战斗参与者状态
type CombatantStatus struct {
    ActorID    model.ID `json:"actor_id"`
    ActorName  string   `json:"actor_name"`
    ActorType  string   `json:"actor_type"`
    HP         int      `json:"hp"`
    MaxHP      int      `json:"max_hp"`
    AC         int      `json:"ac"`
    Conditions []string `json:"conditions"`
    IsDefeated bool     `json:"is_defeated"`
    IsAlly     bool     `json:"is_ally"`
}

// CombatEndState 战斗结束状态
type CombatEndState struct {
    Reason  string `json:"reason"`  // "victory", "defeat", "flee"
    Winners string `json:"winners"` // "players", "enemies"
}
```

增强 `NextTurnResult`：
```go
type NextTurnResult struct {
    Combat *CombatInfo      `json:"combat"`     // 保持向后兼容
    Turn   *EnhancedTurnInfo `json:"turn"`       // 新增：增强回合信息
}
```

内部流程：调用已有 `AdvanceTurn()` → 跳过被突袭角色 → 获取角色信息 → `computeAvailableActions()` → 构建参与者状态列表 → 检测战斗结束条件 → 返回。

**API 3: `ExecuteTurnAction`** — 统一动作执行器

```go
type ExecuteTurnActionRequest struct {
    GameID     model.ID       `json:"game_id"`
    ActorID    model.ID       `json:"actor_id"`
    ActionID   string         `json:"action_id"`   // 来自 AvailableAction.ID
    TargetID   model.ID       `json:"target_id,omitempty"`
    TargetIDs  []model.ID     `json:"target_ids,omitempty"`
    Parameters map[string]any `json:"parameters,omitempty"`
}

type ExecuteTurnActionResult struct {
    Success          bool                   `json:"success"`
    ActionName       string                 `json:"action_name"`
    Narrative        string                 `json:"narrative"` // 结果描述
    AttackResult     *AttackResult          `json:"attack_result,omitempty"`
    DamageResult     *DamageResult          `json:"damage_result,omitempty"`
    HealResult       *HealResult            `json:"heal_result,omitempty"`
    SpellResult      map[string]any         `json:"spell_result,omitempty"`
    MoveResult       *MoveResult            `json:"move_result,omitempty"`
    RemainingActions *AvailableActionsResult `json:"remaining_actions"`
    TurnComplete     bool                   `json:"turn_complete"`
    CombatEnd        *CombatEndState        `json:"combat_end,omitempty"`
    Combat           *CombatInfo            `json:"combat"`
}
```

内部路由：读取 `AvailableAction.Metadata["_route"]` 分派到已有处理器：
- `"attack"` → 调用内部 `executeAttack()` 逻辑
- `"spell"` → 调用内部 `castSpell()` 逻辑
- `"action"` → 调用内部 `executeAction()` 逻辑（dash/dodge等）
- `"move"` → 调用内部 `moveActor()` 逻辑
- `"class_feature"` → 调用职业特性处理器
- `"item"` → 调用物品使用逻辑

每次执行后重新调用 `computeAvailableActions()` 返回剩余可用动作。`RemainingActions` 为空时 `TurnComplete = true`。

---

### Step 3: cdndv2 — 战斗Session核心框架

**目标**：实现 CombatSession 接管机制和战斗循环状态机。

#### 新文件：`game_engine/combat_session.go`

```go
// CombatSession 管理一场完整战斗的独立会话
type CombatSession struct {
    gameID       model.ID
    playerID     model.ID
    engine       *engine.Engine
    llmClient    llm.LLMClient
    registry     *tool.ToolRegistry
    combatAgent  *agent.CombatDMAgent      // 战斗DM Agent
    history      []llm.Message             // 独立对话历史
    state        *CombatLoopState
    logger       *zap.Logger
}

// CombatLoopState 战斗循环状态
type CombatLoopState struct {
    CurrentActorID     model.ID
    CurrentActorType   string  // "player", "enemy", "npc", "scene"
    WaitingForPlayer   bool
    RoundNumber        int
    TurnCount          int
    CombatEnded        bool
    CombatEndReason    string
    LastTurnInfo       *engine.EnhancedTurnInfo // 缓存最近一次turn信息
}

// CombatResult 战斗处理结果
type CombatResult struct {
    Response    string // 返回给玩家的叙述文本
    CombatEnded bool
    Summary     string // 战斗结束时的摘要
}
```

**CombatSession.ProcessInput(ctx, input) 流程**：

```
如果 WaitingForPlayer == true:
  → 将玩家输入交给 CombatDMAgent 解析为合法动作
  → 调用 ExecuteTurnAction 执行
  → 如果 TurnComplete → 自动推进到下一回合
  → 如果仍有动作 → 继续等待玩家输入

如果 WaitingForPlayer == false（首次进入或自动推进）:
  → 调用 NextTurn 获取当前行动者
  → 判断行动者类型：
    player → CombatDMAgent 生成叙述 + 选项展示 → 设 WaitingForPlayer=true → 返回
    enemy  → 创建 EnemyAIAgent → 获取意图 → CombatDMAgent 解析执行 → 继续下一回合
    scene  → 执行场景效果 → 继续下一回合
  → 循环直到遇到玩家回合或战斗结束
```

#### 修改文件：`game_engine/engine.go`

在 `GameSession` 中添加 `combatSession` 字段：

```go
type GameSession struct {
    ID            model.ID
    PlayerID      model.ID
    Engine        *GameEngine
    reactLoop     *ReActLoop
    combatSession *CombatSession  // 新增：战斗时非nil
}
```

修改 `ProcessInput()` 添加战斗拦截逻辑：

```go
func (ge *GameEngine) ProcessInput(ctx context.Context, session *GameSession, input string) (string, error) {
    // 1. 检查是否有活跃的战斗session
    if session.combatSession != nil {
        result, err := session.combatSession.ProcessInput(ctx, input)
        if err != nil { return "", err }

        if result.CombatEnded {
            // 将战斗摘要注入主历史
            session.reactLoop.state.History = append(
                session.reactLoop.state.History,
                llm.NewAssistantMessage(result.Summary, nil),
            )
            session.combatSession = nil // 释放战斗session
        }
        return result.Response, nil
    }

    // 2. 检查游戏phase是否进入了战斗（由之前的MainAgent触发）
    phase, _ := ge.dndEngine.GetPhase(ctx, session.ID)
    if phase == model.PhaseCombat {
        // 创建新的CombatSession并接管
        session.combatSession = NewCombatSession(...)
        result, err := session.combatSession.Initialize(ctx)
        // ... 处理首次进入战斗的逻辑
        return result.Response, nil
    }

    // 3. 正常路径：MainAgent ReActLoop
    // ... 现有逻辑 ...
}
```

---

### Step 4: cdndv2 — 战斗DM Agent

**目标**：实现战斗模式的主Agent，直接持有战斗工具，负责叙事和动作解析。

#### 新文件：`game_engine/agent/combat_dm_agent.go`

`CombatDMAgent` 实现 `Agent` 接口（不是SubAgent），类似于 `MainAgent`：

- **直接持有所有战斗工具**（不需要 delegate_task 间接调用）
- 工具列表：`setup_combat`, `next_turn`(增强版), `execute_turn_action`, `get_current_combat`, `get_actor`, `end_combat`
- 通过LLM function calling 解析玩家自然语言输入为合法动作
- 使用独立的 system prompt 模板

**关键职责**：
1. 根据 `EnhancedTurnInfo` 生成战场描述和选项展示
2. 解析玩家/敌人AI的自然语言输入，映射到 `AvailableAction.ID`
3. 调用 `execute_turn_action` 执行动作
4. 生成战斗叙事（攻击结果、伤害描述等）

#### 新文件：`game_engine/prompt/combat_dm_system.md`

```markdown
# 角色定义
你是D&D 5e战斗DM，负责管理一场完整的回合制战斗。

# 核心职责
1. 以DM口吻描述战场态势和战斗进程
2. 向玩家展示可用动作选项（基于系统提供的available_actions）
3. 解析玩家的自然语言输入，匹配到合法动作并执行
4. 为敌人AI的意图验证合法性并执行

# 当前战斗信息
{{.BattlefieldState}}

# 可用工具
- execute_turn_action: 执行一个动作（攻击、施法、移动等）
- next_turn: 推进到下一个角色的回合
- end_combat: 结束战斗

# 输出规则
- 清晰展示攻击掷骰和伤害数值
- 保持战斗节奏感
- 绝不自行计算伤害或掷骰，一切通过工具执行
```

#### 修改文件：`game_engine/agent/const.go`

```go
const CombatDMAgentName = "combat_dm_agent"
```

---

### Step 5: cdndv2 — 敌人/NPC AI Agent

**目标**：实现临时的角色扮演Agent，为敌人/NPC回合生成行动意图。

#### 新文件：`game_engine/agent/enemy_ai_agent.go`

`EnemyAIAgent` 是一个轻量级、无工具、临时创建的Agent：

```go
type EnemyAIAgent struct {
    llmClient llm.LLMClient
    logger    *zap.Logger
}

// GenerateIntent 根据战场态势生成行动意图
func (a *EnemyAIAgent) GenerateIntent(ctx context.Context, req *EnemyIntentRequest) (string, error)

type EnemyIntentRequest struct {
    ActorName       string                        // 敌人名称
    ActorDescription string                       // 敌人描述/性格
    BattlefieldState string                       // 格式化的战场态势
    AvailableActions *engine.AvailableActionsResult // 可用动作列表
    RecentHistory    []llm.Message                 // 最近几轮战斗历史
}
```

**生命周期**：每个敌人回合开始时创建，调用一次 `GenerateIntent`，获得自然语言意图后销毁。

**返回示例**："我想压低身子从岩石侧面绕过去，趁战士不备用弯刀攻击他的腿，然后滚回掩体。"

战斗DM Agent 收到意图后，通过LLM将其映射到合法的 `AvailableAction.ID` 并执行。

#### 新文件：`game_engine/prompt/enemy_ai_system.md`

```markdown
你是{{.ActorName}}。{{.ActorDescription}}

# 当前战场
{{.BattlefieldState}}

# 你可以做的事
{{.FormattedActions}}

用1-2句话描述你想做什么，说明理由。你不需要执行动作，只需要表达意图。
```

#### 场景AI Agent

场景回合（如火焰地形伤害）逻辑相对简单，可复用 `EnemyAIAgent` 的模式，或在 `CombatSession` 中硬编码常见场景效果，避免不必要的LLM调用。初期实现中，场景效果由 `CombatDMAgent` 直接决策即可。

---

### Step 6: 历史管理与战斗结算

**目标**：实现独立历史、战斗摘要生成、控制权归还。

#### 历史管理策略

**战斗开始时**：
- 从 `ReActLoop.state.History` 提取最近5-10条有意义的 user/assistant 消息
- 压缩为一条 "战前上下文" 系统消息，作为 `combatHistory` 的种子
- 主历史冻结（不再追加消息，但保持引用以便后续追加摘要）

**战斗进行中**：
- 所有战斗相关的 LLM 对话、工具调用、工具结果都写入 `combatHistory`
- 主历史不受影响

**战斗结束时**：
1. 调用LLM，输入完整 `combatHistory`，生成2-5段叙事摘要
2. 摘要内容包括：战斗起因、关键时刻、伤亡情况、最终结果
3. 将摘要作为 assistant 消息追加到 `ReActLoop.state.History`
4. 销毁 `combatHistory` 和 `CombatSession`
5. 下一次 `ProcessInput` 恢复正常 MainAgent 流程

#### 战斗结束检测

在每次 `NextTurn` / `ExecuteTurnAction` 返回时检查 `CombatEndState`：
- 所有敌人被击败 → `{Reason: "victory"}`
- 玩家角色HP归0 → `{Reason: "defeat"}`
- 战斗DM Agent 调用 `end_combat` → 手动结束

---

## 文件清单

### 新建文件（7个）

| 文件路径 | 用途 |
|---------|------|
| `../dnd-core/pkg/engine/available_actions.go` | 可用动作计算引擎 + `GetAvailableActions` API + 类型定义 |
| `../dnd-core/pkg/engine/combat_composite.go` | 3个组合API: `SetupCombat`, 增强`NextTurn`, `ExecuteTurnAction` |
| `game_engine/combat_session.go` | CombatSession 接管机制、战斗循环状态机、历史管理、摘要生成 |
| `game_engine/agent/combat_dm_agent.go` | 战斗DM Agent（实现Agent接口，直接持有战斗工具） |
| `game_engine/agent/enemy_ai_agent.go` | 敌人/NPC AI Agent（临时、无工具、纯LLM推理） |
| `game_engine/prompt/combat_dm_system.md` | 战斗DM系统提示词模板 |
| `game_engine/prompt/enemy_ai_system.md` | 敌人AI系统提示词模板 |

### 修改文件（7个）

| 文件路径 | 修改内容 |
|---------|---------|
| `../dnd-core/pkg/model/combat.go` | `TurnState` 新增 `AttacksRemaining`/`AttacksTotal` 字段 |
| `../dnd-core/pkg/engine/combat.go` | `NextTurnResult` 增加 `Turn *EnhancedTurnInfo` 字段；`combatStateToInfo` 可能需要微调 |
| `game_engine/engine.go` | `GameSession` 添加 `combatSession` 字段；`ProcessInput()` 添加战斗拦截路由 |
| `game_engine/agent/const.go` | 添加 `CombatDMAgentName` 常量 |
| `game_engine/agents.go` | 注册新的组合战斗工具；创建 CombatDMAgent 实例 |
| `game_engine/tool/combat_tools.go` | 添加 `SetupCombatTool`, `ExecuteTurnActionTool` 等新工具包装器 |
| `game_engine/prompt/embed.go`（或prompt包的embed声明） | 确保新的 .md 文件被 `//go:embed` 包含 |

---

## 实现顺序

由于依赖关系，建议严格按步骤顺序实现：

1. **Step 1** (dnd-core: available_actions) → 无外部依赖，纯逻辑
2. **Step 2** (dnd-core: 组合API) → 依赖 Step 1 的类型和计算函数
3. **Step 3** (cdndv2: CombatSession框架) → 依赖 Step 2 的API
4. **Step 4** (cdndv2: CombatDMAgent) → 依赖 Step 3 的框架
5. **Step 5** (cdndv2: EnemyAIAgent) → 依赖 Step 4 的Agent模式
6. **Step 6** (历史管理与结算) → 依赖 Step 3-5 全部完成

每个步骤完成后可独立编译验证。

---

## 验证方案

### 编译验证
```bash
# dnd-core 编译
cd ../dnd-core && go build ./...

# cdndv2 编译
cd /Users/wastecat/code/go/cdndv2 && go build -o cdndv2 .
```

### 单元测试

**dnd-core 侧**：
- `available_actions_test.go`：测试各种状态组合下的动作过滤（昏迷角色返回空、沉默法师无言语法术、法术位耗尽无施法选项等）
- `combat_composite_test.go`：测试 SetupCombat → NextTurn → ExecuteTurnAction 的完整流程

**cdndv2 侧**：
- `combat_session_test.go`：测试战斗拦截路由（phase=combat时进入CombatSession）
- `combat_dm_agent_test.go`：测试Agent的LLM交互（需真实API key）
- `enemy_ai_agent_test.go`：测试意图生成

### 集成测试
```bash
OPENAI_API_KEY=sk-... go test -run TestCombatAgentFullFlow -v -timeout 10m ./game_engine/
```

测试场景：
1. MainAgent 叙事中触发战斗 → 验证 CombatSession 创建
2. 玩家回合：接收战场描述 → 输入动作 → 验证执行结果
3. 敌人回合：验证 EnemyAIAgent 生成意图 → CombatDMAgent 解析执行
4. 战斗结束：验证摘要生成 → 注入主历史 → 控制权归还
5. 归还后继续叙事：验证 MainAgent 正常恢复
