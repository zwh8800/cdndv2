# CombatSession Integration Test

## Context

CombatSession + CombatDMAgent 两层循环架构已实现，但缺少集成测试验证完整战斗流程。此外发现 `NewGameEngine` 中 `enemyAI` 字段未被初始化赋值（声明在 line 49，使用在 line 285，但 return 语句中缺失），需一并修复。

## 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `game_engine/combat_session_test.go` | 新建 | 集成测试 |
| `game_engine/engine.go` | 修复 | 在 `NewGameEngine` 中创建 `enemyAI` 并赋值到返回结构体 |

## Step 1: 修复 engine.go 中 enemyAI 未初始化 bug

在 `NewGameEngine` 的 return 前添加 `enemyAI` 创建，并在 return 语句中赋值：

```go
// 在 mainAgent.SetLogger(logger) 之后添加
enemyAI := agent.NewEnemyAIAgent(llmClient, logger)

// return 语句中添加 enemyAI 字段
return &GameEngine{
    ...
    enemyAI:   enemyAI,
    ...
}, nil
```

## Step 2: 创建 combat_session_test.go

### 测试结构

```
TestCombatSessionFullBattle
  1. Guard: OPENAI_API_KEY 不存在则 t.Skip
  2. 创建 GameEngine（复用 getOpenAIConfigFromEnv）
  3. 通过 dnd-core engine 直接准备数据：
     a. NewGame → gameID
     b. CreatePC → pcID (强力5级战士, HP44)
     c. CreateEnemy → enemyID (弱哥布林, HP7, AC10)
     d. SetPhase → PhaseExploration
     e. SetupCombat → 战斗已初始化
  4. 创建 CombatSession（使用 GameEngine 的内部组件）
  5. cs.Initialize(ctx) → 初始战斗叙述
  6. 循环 cs.ProcessInput(ctx, "我用武器攻击哥布林")
     - 最多 maxRounds=10 轮
     - 每轮记录日志
     - CombatEnded=true 时退出
  7. 记录结果
```

### 角色数据设计

**PC (5级人类战士)** — 强力，确保快速击杀：
- STR 18(+4), DEX 14(+2), CON 16(+3), INT 10, WIS 12, CHA 8
- HP 44, Level 5 (Extra Attack)
- Race: "人类", Class: "战士", Background: "士兵"

**Enemy (哥布林)** — 极弱，1-2 回合可击杀：
- STR 8, DEX 10, CON 8, INT 6, WIS 8, CHA 6
- HP 7, AC 10, AttackBonus 2, DamagePerRound 3
- CR "1/4"

### 断言策略

**Hard assertions** (Fatal on failure):
- 所有 dnd-core 数据准备调用成功
- `Initialize()` 不报错，返回非空 Response
- 每次 `ProcessInput()` 不报错，返回非空 Response

**Soft assertions** (仅 Log):
- 战斗是否在 maxRounds 内结束（LLM + 骰子不确定性，不 Fail）

### 超时

注释说明需要 `-timeout 10m`。每轮 LLM 调用约 2-5 次，每次 35-42 秒。

## Verification

```bash
# 运行测试
OPENAI_API_KEY=sk-... go test -run TestCombatSessionFullBattle -v -timeout 10m ./game_engine/

# 无 API Key 时自动跳过
go test -run TestCombatSessionFullBattle -v ./game_engine/

# 编译检查
go build ./...
go vet ./...
```
