# CombatDMAgent 重设计：自主 Tool-Calling Agent + 框架层回合路由

## Context

当前 CombatDMAgent 只是一个薄工具类，CombatSession 手动编排全部战斗逻辑（720 行）。需要将其重设计为自主 Agent，由 LLM 通过 function calling 驱动战斗工具。

**关键设计决策**：EnemyAIAgent 的调用不是一个 LLM tool，而是 Go 框架层面的逻辑。LLM yield 后，Go 代码根据回合状态判断 yield 给谁。

## 架构：两层循环

```
RunLoop(ctx, input):
  ┌─────────────────────────────────────────────────┐
  │  外层循环 (Go 框架：回合路由)                      │
  │                                                   │
  │  1. 将 input 加入 history                         │
  │  2. 运行内层 ReAct 循环 → LLM yield 出文本        │
  │  3. 检查：战斗结束？→ return (CombatEnded=true)    │
  │  4. 检查：当前是玩家回合？→ return (yield给玩家)    │
  │  5. 当前是敌人回合 → 调用 EnemyAIAgent             │
  │     → 将意图作为 UserMessage 加入 history          │
  │     → 回到步骤 2                                   │
  │                                                   │
  │  ┌───────────────────────────────────────────┐    │
  │  │  内层循环 (LLM ReAct: tool calling)       │    │
  │  │                                           │    │
  │  │  for i < maxIterations:                   │    │
  │  │    resp = LLM(history, tools)             │    │
  │  │    if resp.ToolCalls:                     │    │
  │  │      execute tools → add to history       │    │
  │  │      update turnState from results        │    │
  │  │      continue                             │    │
  │  │    else:                                  │    │
  │  │      add text to history → YIELD          │    │
  │  │      return resp.Content                  │    │
  │  └───────────────────────────────────────────┘    │
  └─────────────────────────────────────────────────┘
```

**LLM 的系统提示词指导它**：调用工具执行动作后，叙述结果，然后 STOP。无论是玩家回合还是敌人回合，LLM 都在 `next_turn_with_actions` 调用后描述当前态势然后 yield。

**Go 框架在 LLM yield 后**：
- 解析缓存的回合信息（从 `next_turn_with_actions` 工具结果中提取）
- 若当前角色是 PC → 真正 yield，返回给 `ProcessInput` 调用方
- 若当前角色是 Enemy/NPC → 调用 `EnemyAIAgent.GenerateIntent()` → 将意图作为新的 `UserMessage` 喂回 history → 重新进入内层 ReAct 循环

从 LLM 的视角看，敌人的意图和玩家的输入完全一样——都是 UserMessage。

## 实施步骤

### Step 1: 创建 CombatDM 专用工具 (`game_engine/tool/combat_dm_tools.go`)

3 个新工具（没有 `get_enemy_intent`）：

| 工具名 | 包装的 API | ReadOnly | 说明 |
|--------|-----------|----------|------|
| `execute_turn_action` | `engine.ExecuteTurnAction()` | false | 执行战斗动作，返回 narrative + remaining_actions + turn_complete + combat_end |
| `next_turn_with_actions` | `engine.NextTurnWithActions()` | false | 推进回合，返回 EnhancedTurnInfo（角色信息 + 可用动作 + 参与者状态 + combat_end） |
| `get_available_actions` | `engine.GetAvailableActions()` | true | 查询角色可用动作列表 |

### Step 2: 重写 CombatDMAgent (`game_engine/agent/combat_dm_agent.go`)

```go
type CombatDMAgent struct {
    gameID    model.ID
    playerID  model.ID
    llmClient llm.LLMClient
    registry  *tool.ToolRegistry
    enemyAI   *EnemyAIAgent
    history   []llm.Message     // 持久化对话历史
    logger    *zap.Logger

    // 从工具结果中缓存的回合状态
    turnState *turnStateCache
    ended     bool
}

type turnStateCache struct {
    ActorID    model.ID
    ActorName  string
    ActorType  string // "pc", "enemy", "npc", "companion"
    TurnInfo   json.RawMessage // 缓存的 EnhancedTurnInfo JSON
}
```

**核心方法**：

`RunLoop(ctx, input) → *CombatAgentResult`：
```
func RunLoop(ctx, input):
  if input != "": history.append(UserMessage(input))
  
  for {  // 外层循环：回合路由
    response := executeReActLoop(ctx)  // 内层 LLM 循环，直到 yield
    
    if ended:
      return {Response: response, CombatEnded: true}
    
    if isPlayerTurn():
      return {Response: response, CombatEnded: false}  // yield 给玩家
    
    // 敌人/NPC 回合 → 生成意图，喂回 LLM
    intent := generateEnemyIntent(ctx)
    history.append(UserMessage(
      fmt.Sprintf("[%s的行动] %s", turnState.ActorName, intent)
    ))
    // 回到外层循环顶部 → 再次运行 executeReActLoop
  }
```

`executeReActLoop(ctx) → string`：内层 LLM tool-calling 循环
```
func executeReActLoop(ctx):
  for i := 0; i < 20; i++:
    resp = llmClient.Complete(ctx, {Messages: history, Tools: toolSchemas})
    
    if len(resp.ToolCalls) > 0:
      history.append(AssistantMessage(resp.Content, resp.ToolCalls))
      results = registry.ExecuteTools(ctx, resp.ToolCalls)
      for r in results: history.append(ToolMessage(r.Content, r.ToolCallID))
      updateTurnStateFromResults(results)  // 解析工具结果，更新缓存
      continue
    
    history.append(AssistantMessage(resp.Content, nil))
    return resp.Content  // LLM yield
  
  return "（达到最大迭代次数）"
```

`updateTurnStateFromResults(results)`：扫描工具结果，从 `next_turn_with_actions` 和 `execute_turn_action` 的 JSON 输出中提取 ActorType/ActorID/ActorName 等信息，更新 `turnState` 缓存。同时检测 `combat_end` 字段设置 `ended` flag。

`generateEnemyIntent(ctx) → string`：使用缓存的 turnState 构建上下文，调用 `enemyAI.GenerateIntent()`。

`isPlayerTurn() → bool`：`turnState.ActorType == "pc"`

`Initialize(ctx)` → 添加系统提示词到 history → 运行 RunLoop(ctx, "")

### Step 3: 重写系统提示词 (`game_engine/prompt/combat_dm_system.md`)

关键指令：

```markdown
# 角色
你是 D&D 5e 战斗 DM，通过工具调用自主管理战斗。

# 会话信息
- 游戏ID: {{.GameID}}
- 玩家ID: {{.PlayerID}}

# 可用工具
{{range .AvailableTools}}
- `{{.Name}}`: {{.Description}}
{{end}}

# 工作流
1. 收到输入后，理解意图，调用适当的工具执行
2. 根据工具结果生成叙事描述
3. 如果 turn_complete=true，调用 next_turn_with_actions 推进回合
4. 描述新的回合态势后 STOP（不再调用工具）
5. 等待下一个输入（可能是玩家，也可能是敌人的行动描述）

# 输入格式
- 玩家输入：自然语言（"我用长剑攻击地精"）
- 敌人/NPC行动：格式为"[角色名的行动] 意图描述"

# 叙事规则
- 使用工具结果中的数值（骰点、伤害等）融入叙事
- 玩家角色用第二人称，敌人/NPC用第三人称
- 保持紧凑（2-4句）
- 每次 STOP 前必须描述当前局势和可用选项

# 禁止行为
- 不得自行计算攻击/伤害/检定
- 不得跳过调用工具直接叙述结果
- 不得在一次响应中处理多个回合的输入
```

### Step 4: 调整 EnemyAIAgent (`game_engine/agent/enemy_ai_agent.go`)

- **保留** `GenerateIntent()` — 被框架层直接调用
- **删除** `ResolveIntent()` — CombatDMAgent 的 LLM 自行映射意图到 action_id
- **保留** `SystemPrompt()` 和 `enemy_ai_system.md`

### Step 5: 简化 CombatSession (`game_engine/combat_session.go`)

从 720 行 → ~80 行薄包装器：

```go
type CombatSession struct {
    gameID    model.ID
    playerID  model.ID
    combatDM  *CombatDMAgent
    logger    *zap.Logger
}

func NewCombatSession(gameID, playerID, engine, llmClient, registry, enemyAI, logger) *CombatSession

func (cs *CombatSession) Initialize(ctx) (*CombatResult, error)
  → combatDM.Initialize(ctx)

func (cs *CombatSession) ProcessInput(ctx, input) (*CombatResult, error)
  → combatDM.RunLoop(ctx, input)

func (cs *CombatSession) IsWaitingForPlayer() bool
  → !combatDM.ended
```

**删除全部手动编排代码**（~600 行）：processCurrentTurn, handlePlayerTurn, handleEnemyTurn, advanceToNextTurn, resolvePlayerInput, fuzzyMatchAction, generateEnemyIntent, resolveIntentToAction, pickDefaultEnemyAction, buildBattlefieldDescription, formatAvailableActions, formatSingleAction, buildActionNarrative, handleCombatEnd, generateCombatSummary 等。

### Step 6: 注册工具 (`game_engine/agents.go`)

```go
// ========== CombatDM Agent 专用工具 ==========
registry.Register(tool.NewExecuteTurnActionTool(engine), []string{agent.CombatDMAgentName}, "combat_dm")
registry.Register(tool.NewNextTurnWithActionsTool(engine), []string{agent.CombatDMAgentName}, "combat_dm")
registry.Register(tool.NewGetAvailableActionsTool(engine), []string{agent.CombatDMAgentName}, "combat_dm")
```

同时把已有只读工具注册给 CombatDMAgent：
- `get_current_combat`, `get_current_turn` — 添加 `CombatDMAgentName`
- `end_combat` — 添加 `CombatDMAgentName`
- `get_actor` — 添加 `CombatDMAgentName`

### Step 7: 更新 engine.go

1. `NewCombatSession()` 新增 `registry` 和 `enemyAI` 参数
2. 在 `NewGameEngine()` 中创建 `EnemyAIAgent` 实例并保存
3. 战斗拦截逻辑传入 registry 和 enemyAI

## 关键文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `game_engine/tool/combat_dm_tools.go` | **新建** | 3 个工具包装器 |
| `game_engine/agent/combat_dm_agent.go` | **重写** | 自主 Agent + 两层循环 |
| `game_engine/prompt/combat_dm_system.md` | **重写** | 带工具调用工作流的系统提示词 |
| `game_engine/agent/enemy_ai_agent.go` | **修改** | 删除 ResolveIntent |
| `game_engine/combat_session.go` | **重写** | 720→~80 行薄包装器 |
| `game_engine/agents.go` | **修改** | 注册 3 个新工具 + 复用已有工具 |
| `game_engine/engine.go` | **修改** | 传 registry/enemyAI，创建 EnemyAIAgent |

## 边界情况

- **内层 maxIterations (20)**：LLM 连续工具调用次数上限
- **外层安全阀**：连续 10 个敌人回合后强制 yield（防止无限敌人链）
- **LLM 调用失败**：重试一次，仍失败返回错误，不污染 history
- **工具执行失败**：错误作为 ToolMessage 加入 history，LLM 看到后调整
- **turnState 解析失败**：回退为 yield 给玩家（保守策略）
- **战斗结束检测**：从 `execute_turn_action` 和 `next_turn_with_actions` 结果中检测 `combat_end` 字段
- **历史过长（>100 条）**：保留 system prompt + 最近 30 条，中间压缩为摘要

## 验证

```bash
cd ../dnd-core && go build ./... && go vet ./...
cd ../cdndv2 && go build ./... && go vet ./...
```
