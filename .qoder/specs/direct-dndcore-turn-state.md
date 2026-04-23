# 直接调用 dnd-core 获取回合状态，替换 JSON 解析

## Context

CombatDMAgent 当前通过解析 LLM 工具返回的 JSON 字符串来提取回合状态（谁在行动、actor 类型、战斗是否结束）。这种方式脆弱且不必要——agent 持有 gameID，完全可以直接调用 dnd-core 的只读查询 API 获取精确的游戏状态。

同理，`generateEnemyIntent` 从 LLM 对话历史中提取战场信息和可用动作文本，也应替换为直接 API 调用。

## 方案

### 1. 给 CombatDMAgent 注入 dnd-core Engine

**文件**: `game_engine/agent/combat_dm_agent.go`

- 新增 `dndEngine` 字段（类型 `*engine.Engine`，import `github.com/zwh8800/dnd-core/pkg/engine`）
- 构造函数 `NewCombatDMAgent` 增加 `dndEngine` 参数
- 新增 `cachedActions *engine.AvailableActionsResult` 字段，供 EnemyAI 使用

### 2. 透传 engine 到创建链

**文件**: `game_engine/combat_session.go`
- `NewCombatSession` 增加 `dndEngine *engine.Engine` 参数，传给 `NewCombatDMAgent`

**文件**: `game_engine/engine.go`
- 创建 `CombatSession` 时传入 `ge.dndEngine`

**文件**: `game_engine/combat_session_test.go`
- 测试中传入 dnd-core engine

### 3. 新增 `refreshTurnState(ctx)` 方法

替换 `updateTurnStateFromResults` + `tryParseTurnState` + `extractJSON`。

逻辑：
```go
func (a *CombatDMAgent) refreshTurnState(ctx context.Context) {
    // 1. 调用 GetCurrentCombat 检查战斗是否活跃
    combatResult, err := a.dndEngine.GetCurrentCombat(ctx, engine.GetCurrentCombatRequest{
        GameID: a.gameID,
    })
    if err != nil || combatResult.Combat.Status != model.CombatStatusActive {
        a.ended = true
        return
    }

    // 2. 从 CurrentTurn 获取当前行动者 ID
    actorID := combatResult.Combat.CurrentTurn.ActorID
    actorName := combatResult.Combat.CurrentTurn.ActorName

    // 3. 调用 GetAvailableActions 获取 ActorType + 可用动作
    actions, err := a.dndEngine.GetAvailableActions(ctx, engine.GetAvailableActionsRequest{
        GameID:  a.gameID,
        ActorID: actorID,
    })
    if err != nil {
        // fallback: 仅用 combat info 中的信息
        a.turnState = &turnStateCache{ActorID: actorID, ActorName: actorName}
        return
    }

    a.turnState = &turnStateCache{
        ActorID:   actorID,
        ActorName: actions.ActorName,
        ActorType: actions.ActorType,
    }
    a.cachedActions = actions
}
```

在 ReAct 循环中，工具执行后调用 `a.refreshTurnState(ctx)` 替换原来的 `a.updateTurnStateFromResults(results)`。

### 4. 新增 `formatBattlefieldSummary(ctx)` 方法

替换 `extractRecentBattlefield`（原方案从历史 assistant 消息中取叙述文本）。

逻辑：
```go
func (a *CombatDMAgent) formatBattlefieldSummary(ctx context.Context) string {
    summary, err := a.dndEngine.GetStateSummary(ctx, a.gameID)
    if err != nil || summary.ActiveCombat == nil {
        return "战斗进行中"
    }
    // 格式化: 第X轮，各参战者 HP/AC/状态
    var sb strings.Builder
    combat := summary.ActiveCombat
    sb.WriteString(fmt.Sprintf("第%d轮，当前行动者: %s\n", combat.Round, combat.CurrentActor))
    for _, c := range combat.Combatants {
        status := ""
        if c.IsDefeated { status = " [已倒下]" }
        if len(c.Conditions) > 0 { status += " " + strings.Join(c.Conditions, ",") }
        sb.WriteString(fmt.Sprintf("- %s (%s): HP %d/%d, AC %d%s\n",
            c.Name, c.Type, c.HP, c.MaxHP, c.AC, status))
    }
    return sb.String()
}
```

### 5. 新增 `formatAvailableActions()` 方法

替换 `extractAvailableActions`（原方案从历史 tool 消息中搜索含 "available_actions" 的内容）。

逻辑：
```go
func (a *CombatDMAgent) formatAvailableActions() string {
    if a.cachedActions == nil {
        return "（可用动作信息不可用）"
    }
    // 格式化所有可用动作
    var sb strings.Builder
    formatActions := func(label string, actions []engine.AvailableAction) {
        if len(actions) == 0 { return }
        sb.WriteString(label + ":\n")
        for _, act := range actions {
            desc := act.Name
            if act.Range != "" { desc += " (" + act.Range + ")" }
            if act.DamagePreview != "" { desc += " " + act.DamagePreview }
            sb.WriteString(fmt.Sprintf("  - %s: %s\n", act.ID, desc))
        }
    }
    if a.cachedActions.Movement.Available {
        sb.WriteString(fmt.Sprintf("移动: 剩余 %d 尺\n", a.cachedActions.Movement.RemainingFeet))
    }
    formatActions("动作", a.cachedActions.Actions)
    formatActions("附赠动作", a.cachedActions.BonusActions)
    formatActions("自由动作", a.cachedActions.FreeActions)
    return sb.String()
}
```

### 6. 更新 `generateEnemyIntent`

```go
func (a *CombatDMAgent) generateEnemyIntent(ctx context.Context) (string, error) {
    if a.turnState == nil {
        return "攻击最近的敌人", nil
    }
    battlefield := a.formatBattlefieldSummary(ctx)
    actionsText := a.formatAvailableActions()
    return a.enemyAI.GenerateIntent(ctx, a.turnState.ActorName, a.turnState.ActorType, battlefield, actionsText)
}
```

### 7. 删除以下函数

- `updateTurnStateFromResults(results []llm.ToolResult)`
- `tryParseTurnState(content string)`
- `extractJSON(content string) string`
- `extractRecentBattlefield() string`
- `extractAvailableActions() string`

## 修改文件清单

| 文件 | 变更 |
|------|------|
| `game_engine/agent/combat_dm_agent.go` | 核心重构：新增 dndEngine 字段、refreshTurnState、formatBattlefieldSummary、formatAvailableActions；删除5个旧函数 |
| `game_engine/combat_session.go` | NewCombatSession 增加 dndEngine 参数 |
| `game_engine/engine.go` | 创建 CombatSession 时传入 dndEngine |
| `game_engine/combat_session_test.go` | 更新测试构造函数调用 |

## 验证

```bash
go vet ./game_engine/...
go build ./...
OPENAI_API_KEY=sk-... go test -run TestCombatSessionFullBattle -v -timeout 10m ./game_engine/
```
