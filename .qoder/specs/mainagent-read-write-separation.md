# Spec: MainAgent 读写分离 - 只读直调 / 写操作委托

## 背景

当前 MainAgent 拥有所有工具的直接调用权限（通过 `registry.GetAllTools()`），这与 SubAgent 委托机制的目标相矛盾：
- 写操作（造角色、进战斗、施法等）绕过 SubAgent 直接执行，失去了会话隔离、并行执行、结果合成等改进
- LLM 面对所有工具时容易混淆，不确定该直调还是委托

本 spec 实现"读操作直调、写操作委托"的分层收窄策略。

## 设计原则

**读操作可以直调，写操作必须委托。**

- **只读工具 (ReadOnly)**：查询游戏状态，不修改任何数据。MainAgent 可直接调用，效率高、延迟低
- **写操作工具 (Write)**：修改游戏状态（创建/删除/更新/执行）。必须通过 `delegate_task` 委托给 SubAgent，享受会话隔离、并行执行、结果合成等机制

## 工具分类

### 只读工具 (ReadOnly) - MainAgent 可直调

| 工具名 | 分类 | 说明 |
|--------|------|------|
| `get_actor` | character | 查询角色信息 |
| `get_pc` | character | 查询玩家角色信息 |
| `list_actors` | character | 列出所有角色 |
| `get_current_combat` | combat | 查询当前战斗状态 |
| `get_current_turn` | combat | 查询当前回合 |
| `get_passive_perception` | check | 查询被动感知 |
| `get_spell_slots` | spell | 查询法术位 |

### 写操作工具 (Write) - 必须委托 SubAgent

| 工具名 | 分类 | 委托目标 |
|--------|------|----------|
| `create_pc` | character | character_agent |
| `create_npc` | character | character_agent |
| `create_enemy` | character | character_agent |
| `create_companion` | character | character_agent |
| `update_actor` | character | character_agent |
| `remove_actor` | character | character_agent |
| `add_experience` | character | character_agent |
| `start_combat` | combat | combat_agent |
| `start_combat_with_surprise` | combat | combat_agent |
| `next_turn` | combat | combat_agent |
| `execute_action` | combat | combat_agent |
| `execute_attack` | combat | combat_agent |
| `move_actor` | combat | combat_agent |
| `execute_damage` | combat | combat_agent |
| `execute_healing` | combat | combat_agent |
| `perform_death_save` | combat | combat_agent |
| `end_combat` | combat | combat_agent |
| `perform_ability_check` | check | rules_agent |
| `perform_skill_check` | check | rules_agent |
| `perform_saving_throw` | check | rules_agent |
| `short_rest` | rest | rules_agent |
| `cast_spell` | spell | rules_agent |
| `prepare_spells` | spell | rules_agent |
| `learn_spell` | spell | rules_agent |
| `concentration_check` | spell | rules_agent |
| `end_concentration` | spell | rules_agent |
| `start_long_rest` | rest | rules_agent |
| `end_long_rest` | rest | rules_agent |

## 实现步骤

### Step 1: Tool 接口添加 ReadOnly 标记

**文件**: `game_engine/tool/tool.go`

在 `Tool` 接口中新增 `ReadOnly()` 方法：

```go
// Tool 基础接口
type Tool interface {
    Name() string
    Description() string
    ParametersSchema() map[string]any
    Execute(ctx context.Context, params map[string]any) (*ToolResult, error)
    ReadOnly() bool  // 新增：标记是否为只读工具
}
```

在 `BaseTool` 中添加 `readOnly` 字段和默认实现：

```go
type BaseTool struct {
    name        string
    description string
    schema      map[string]any
    readOnly    bool  // 新增
}

// ReadOnly 返回是否为只读工具（默认 false）
func (t *BaseTool) ReadOnly() bool {
    return t.readOnly
}
```

新增 `NewBaseTool` 构造函数（保留 `NewEngineTool` 不变）：

```go
func NewBaseTool(name, description string, schema map[string]any, readOnly bool) BaseTool {
    return BaseTool{
        name:        name,
        description: description,
        schema:      schema,
        readOnly:    readOnly,
    }
}
```

`EngineTool` 同步添加 `readOnly` 参数：

```go
type EngineTool struct {
    BaseTool
    engine any
}

func NewEngineTool(name, description string, schema map[string]any, engine any, readOnly bool) *EngineTool {
    return &EngineTool{
        BaseTool: NewBaseTool(name, description, schema, readOnly),
        engine:   engine,
    }
}
```

### Step 2: 更新所有工具构造函数，传入 ReadOnly 标记

**文件**: `game_engine/tool/character_tools.go`

修改所有 `NewXxxTool` 函数，只读工具传入 `true`，写操作工具传入 `false`：

- `NewCreatePCTool` → `NewEngineTool(..., engine, false)`
- `NewCreateNPCTool` → `NewEngineTool(..., engine, false)`
- `NewCreateEnemyTool` → `NewEngineTool(..., engine, false)`
- `NewCreateCompanionTool` → `NewEngineTool(..., engine, false)`
- `NewGetActorTool` → `NewEngineTool(..., engine, true)`
- `NewGetPCTool` → `NewEngineTool(..., engine, true)`
- `NewListActorsTool` → `NewEngineTool(..., engine, true)`
- `NewUpdateActorTool` → `NewEngineTool(..., engine, false)`
- `NewRemoveActorTool` → `NewEngineTool(..., engine, false)`
- `NewAddExperienceTool` → `NewEngineTool(..., engine, false)`

**文件**: `game_engine/tool/combat_tools.go`

- `NewStartCombatTool` → `false`
- `NewStartCombatWithSurpriseTool` → `false`
- `NewGetCurrentCombatTool` → `true`
- `NewGetCurrentTurnTool` → `true`
- `NewNextTurnTool` → `false`
- `NewExecuteActionTool` → `false`
- `NewExecuteAttackTool` → `false`
- `NewMoveActorTool` → `false`
- `NewExecuteDamageTool` → `false`
- `NewExecuteHealingTool` → `false`
- `NewPerformDeathSaveTool` → `false`
- `NewEndCombatTool` → `false`

**文件**: `game_engine/tool/rules_tools.go`

- `NewPerformAbilityCheckTool` → `false`
- `NewPerformSkillCheckTool` → `false`
- `NewPerformSavingThrowTool` → `false`
- `NewGetPassivePerceptionTool` → `true`
- `NewShortRestTool` → `false`
- `NewCastSpellTool` → `false`
- `NewGetSpellSlotsTool` → `true`
- `NewPrepareSpellsTool` → `false`
- `NewLearnSpellTool` → `false`
- `NewConcentrationCheckTool` → `false`
- `NewEndConcentrationTool` → `false`
- `NewStartLongRestTool` → `false`
- `NewEndLongRestTool` → `false`

**文件**: `game_engine/tool/delegate_task_tool.go`

`DelegateTaskTool` 不修改，保持 `BaseTool` 的默认 `ReadOnly() = false`。此工具在 `parseResponse` 中被拦截，不会走常规执行路径。

### Step 3: ToolRegistry 添加按 ReadOnly 过滤的方法

**文件**: `game_engine/tool/registry.go`

新增两个方法：

```go
// GetReadOnlyTools 获取所有只读工具
func (r *ToolRegistry) GetReadOnlyTools() []Tool {
    var tools []Tool
    for _, t := range r.tools {
        if t.ReadOnly() {
            tools = append(tools, t)
        }
    }
    return tools
}

// GetReadOnlySchemas 获取只读工具的 LLM 函数调用格式 Schema
func (r *ToolRegistry) GetReadOnlySchemas() []map[string]any {
    schemas := make([]map[string]any, 0)
    for _, t := range r.tools {
        if t.ReadOnly() {
            schemas = append(schemas, map[string]any{
                "type": "function",
                "function": map[string]any{
                    "name":        t.Name(),
                    "description": t.Description(),
                    "parameters":  t.ParametersSchema(),
                },
            })
        }
    }
    return schemas
}
```

### Step 4: MainAgent 只暴露只读工具 + delegate_task 给 LLM

**文件**: `game_engine/agent/main_agent.go`

修改 `Execute()` 方法中传给 LLM 的 tools 定义。只传只读工具的 schema + delegate_task：

```go
// 获取Tools定义 - 只暴露只读工具和 delegate_task
tools := m.registry.GetReadOnlySchemas()
// 追加 delegate_task 工具
delegateSchema := m.getDelegateTaskSchema()
tools = append(tools, delegateSchema)
```

新增辅助方法：

```go
// getDelegateTaskSchema 获取 delegate_task 工具的 Schema
func (m *MainAgent) getDelegateTaskSchema() map[string]any {
    if t, ok := m.registry.Get(tool.DelegateTaskToolName); ok {
        return map[string]any{
            "type": "function",
            "function": map[string]any{
                "name":        t.Name(),
                "description": t.Description(),
                "parameters":  t.ParametersSchema(),
            },
        }
    }
    return nil
}
```

修改 `Tools()` 方法，返回 MainAgent 实际可用的工具集合：

```go
// Tools 返回Agent可用的Tools（只读工具 + delegate_task）
func (m *MainAgent) Tools() []tool.Tool {
    readOnlyTools := m.registry.GetReadOnlyTools()
    if dt, ok := m.registry.Get(tool.DelegateTaskToolName); ok {
        readOnlyTools = append(readOnlyTools, dt)
    }
    return readOnlyTools
}
```

### Step 5: parseResponse 拦截写操作工具调用

**文件**: `game_engine/agent/main_agent.go`

在 `parseResponse()` 中，即使 LLM 错误地调用了写操作工具（理论上不应发生，但需防御），也要将其转为 delegate_task：

```go
// separateDelegateCalls 改为 separateCallsByAccess
// 将工具调用分为三类：delegate_task 调用、只读工具调用、写操作工具调用
func (m *MainAgent) separateCallsByAccess(toolCalls []llm.ToolCall) (delegateCalls, readOnlyCalls, writeCalls []llm.ToolCall) {
    for _, call := range toolCalls {
        if tool.IsDelegateTaskTool(call.Name) {
            delegateCalls = append(delegateCalls, call)
            continue
        }
        t, ok := m.registry.Get(call.Name)
        if !ok {
            // 未知工具，保留原样让 ToolRegistry 返回错误
            readOnlyCalls = append(readOnlyCalls, call)
            continue
        }
        if t.ReadOnly() {
            readOnlyCalls = append(readOnlyCalls, call)
        } else {
            writeCalls = append(writeCalls, call)
        }
    }
    return
}
```

修改 `parseResponse` 中的处理逻辑：

```go
// 处理Tool调用
if len(resp.ToolCalls) > 0 {
    delegateCalls, readOnlyCalls, writeCalls := m.separateCallsByAccess(resp.ToolCalls)

    // 写操作调用转为 delegate_task
    if len(writeCalls) > 0 {
        m.getLogger().Warn("LLM attempted to call write tools directly, converting to delegate_task",
            zap.Int("writeCallCount", len(writeCalls)),
        )
        for _, wc := range writeCalls {
            agentName := m.inferAgentForTool(wc.Name)
            delegateCalls = append(delegateCalls, llm.ToolCall{
                ID:   wc.ID,
                Name: tool.DelegateTaskToolName,
                Arguments: map[string]any{
                    "agent_name": agentName,
                    "intent":     fmt.Sprintf("执行 %s 操作", wc.Name),
                },
            })
        }
    }

    // 处理 delegate_task 调用
    if len(delegateCalls) > 0 {
        agentResp.SubAgentCalls = m.convertToSubAgentCalls(delegateCalls)
        // 只读工具调用也保留，和委托并行执行
        agentResp.ToolCalls = readOnlyCalls
        if len(delegateCalls) > 0 {
            agentResp.NextAction = ActionDelegate
        } else {
            agentResp.NextAction = ActionContinue
        }
        return agentResp, nil
    }

    // 只有只读工具调用
    agentResp.ToolCalls = readOnlyCalls
    agentResp.NextAction = ActionContinue
    return agentResp, nil
}
```

新增推断工具所属 Agent 的方法：

```go
// inferAgentForTool 根据工具名推断应该委托给哪个 Agent
func (m *MainAgent) inferAgentForTool(toolName string) string {
    // 查询 registry 中的 byAgent 映射
    agents := m.registry.GetAgentsForTool(toolName)
    // 优先返回非 MainAgent 的 Agent
    for _, a := range agents {
        if a != MainAgentName {
            return a
        }
    }
    // 默认委托给 rules_agent
    return SubAgentNameRules
}
```

### Step 6: ToolRegistry 添加 GetAgentsForTool 方法

**文件**: `game_engine/tool/registry.go`

```go
// GetAgentsForTool 获取工具所属的Agent列表
func (r *ToolRegistry) GetAgentsForTool(toolName string) []string {
    var agents []string
    for agent, tools := range r.byAgent {
        for _, t := range tools {
            if t == toolName {
                agents = append(agents, agent)
                break
            }
        }
    }
    return agents
}
```

### Step 7: 更新 agents.go 中工具注册 - 移除 MainAgent 对写工具的关联

**文件**: `game_engine/agents.go`

修改 `registerAgentTools()`，从写操作工具的 agents 列表中移除 `agent.MainAgentName`。

只读工具保留 `agent.MainAgentName`，写操作工具只关联 SubAgent。

修改前：
```go
registry.Register(tool.NewCreatePCTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "character")
```

修改后：
```go
registry.Register(tool.NewCreatePCTool(engine), []string{agent.SubAgentNameCharacter}, "character")
```

对所有写操作工具执行此变更。只读工具保持不变。

完整对照：

```go
func registerAgentTools(registry *tool.ToolRegistry, engine *engine.Engine) {
    // 委托任务工具（MainAgent专用）
    registry.Register(tool.NewDelegateTaskTool(), []string{agent.MainAgentName}, "delegation")

    // ========== 角色管理工具 ==========
    // 写操作 - 仅 SubAgent
    registry.Register(tool.NewCreatePCTool(engine), []string{agent.SubAgentNameCharacter}, "character")
    registry.Register(tool.NewCreateNPCTool(engine), []string{agent.SubAgentNameCharacter}, "character")
    registry.Register(tool.NewCreateEnemyTool(engine), []string{agent.SubAgentNameCharacter, agent.SubAgentNameCombat}, "character")
    registry.Register(tool.NewCreateCompanionTool(engine), []string{agent.SubAgentNameCharacter}, "character")
    registry.Register(tool.NewUpdateActorTool(engine), []string{agent.SubAgentNameCharacter}, "character")
    registry.Register(tool.NewRemoveActorTool(engine), []string{agent.SubAgentNameCharacter}, "character")
    registry.Register(tool.NewAddExperienceTool(engine), []string{agent.SubAgentNameCharacter}, "character")
    // 只读 - MainAgent + SubAgent
    registry.Register(tool.NewGetActorTool(engine), []string{agent.SubAgentNameCharacter, agent.SubAgentNameCombat, agent.SubAgentNameRules, agent.MainAgentName}, "character")
    registry.Register(tool.NewGetPCTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "character")
    registry.Register(tool.NewListActorsTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "character")

    // ========== 战斗系统工具 ==========
    // 写操作 - 仅 SubAgent
    registry.Register(tool.NewStartCombatTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    registry.Register(tool.NewStartCombatWithSurpriseTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    registry.Register(tool.NewNextTurnTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    registry.Register(tool.NewExecuteActionTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    registry.Register(tool.NewExecuteAttackTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    registry.Register(tool.NewMoveActorTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    registry.Register(tool.NewExecuteDamageTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    registry.Register(tool.NewExecuteHealingTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    registry.Register(tool.NewPerformDeathSaveTool(engine), []string{agent.SubAgentNameCombat, agent.SubAgentNameRules}, "combat")
    registry.Register(tool.NewEndCombatTool(engine), []string{agent.SubAgentNameCombat}, "combat")
    // 只读 - MainAgent + SubAgent
    registry.Register(tool.NewGetCurrentCombatTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "combat")
    registry.Register(tool.NewGetCurrentTurnTool(engine), []string{agent.SubAgentNameCombat, agent.MainAgentName}, "combat")

    // ========== 规则检定工具 ==========
    // 写操作 - 仅 SubAgent
    registry.Register(tool.NewPerformAbilityCheckTool(engine), []string{agent.SubAgentNameRules}, "check")
    registry.Register(tool.NewPerformSkillCheckTool(engine), []string{agent.SubAgentNameRules}, "check")
    registry.Register(tool.NewPerformSavingThrowTool(engine), []string{agent.SubAgentNameRules}, "check")
    registry.Register(tool.NewShortRestTool(engine), []string{agent.SubAgentNameRules}, "rest")
    registry.Register(tool.NewCastSpellTool(engine), []string{agent.SubAgentNameRules}, "spell")
    registry.Register(tool.NewPrepareSpellsTool(engine), []string{agent.SubAgentNameRules}, "spell")
    registry.Register(tool.NewLearnSpellTool(engine), []string{agent.SubAgentNameRules}, "spell")
    registry.Register(tool.NewConcentrationCheckTool(engine), []string{agent.SubAgentNameRules}, "spell")
    registry.Register(tool.NewEndConcentrationTool(engine), []string{agent.SubAgentNameRules}, "spell")
    registry.Register(tool.NewStartLongRestTool(engine), []string{agent.SubAgentNameRules}, "rest")
    registry.Register(tool.NewEndLongRestTool(engine), []string{agent.SubAgentNameRules}, "rest")
    // 只读 - MainAgent + SubAgent
    registry.Register(tool.NewGetPassivePerceptionTool(engine), []string{agent.SubAgentNameRules, agent.MainAgentName}, "check")
    registry.Register(tool.NewGetSpellSlotsTool(engine), []string{agent.SubAgentNameRules, agent.MainAgentName}, "spell")
}
```

### Step 8: 更新 MainAgent 的提示词

**文件**: `game_engine/prompt/main_system.md`

更新"工作流程"和"可用能力"部分，明确读写分离规则：

```markdown
# 可用能力

## 可用Tools（只读查询）
以下工具你可以直接调用，它们不会修改游戏状态，仅用于查询信息：

{{range .ReadOnlyTools}}
- `{{.Name}}`: {{.Description}}
{{end}}

## 委托任务

当你需要执行修改游戏状态的操作时，必须使用 `delegate_task` 工具将任务委托给专门的Agent：

- **character_agent**: 角色管理专家 - 负责角色创建、更新、删除、经验
- **combat_agent**: 战斗管理专家 - 负责战斗初始化、回合管理、攻击、伤害、治疗
- **rules_agent**: 规则仲裁专家 - 负责检定、豁免、法术、专注、休息

**使用方式**: 调用 `delegate_task` 工具，指定 agent_name 和 intent 参数。

**示例**:
- 玩家说"我创建一个精灵法师" → `delegate_task(agent_name="character_agent", intent="创建1级精灵法师")`
- 玩家说"我要攻击地精" → `delegate_task(agent_name="combat_agent", intent="攻击地精")`
- 玩家说"我过个感知豁免" → `delegate_task(agent_name="rules_agent", intent="进行感知豁免检定")`

**重要**: 你不能直接调用写操作工具（如创建角色、发起战斗、施法等），必须通过 `delegate_task` 委托。

# 工作流程

1. 分析玩家输入，理解意图
2. 判断任务的类型：
   - **信息查询**: 直接调用只读Tool（如查HP、查战斗状态、查法术位）
   - **状态修改**: 使用 `delegate_task` 委托给对应Agent（如创建角色、攻击、施法）
   - **纯叙事**: 直接生成叙事内容（如描述场景、对话）
3. 执行调用并等待结果
4. 基于结果生成叙事响应
5. 引导玩家下一步行动
```

### Step 9: 更新 prepareTemplateData

**文件**: `game_engine/agent/main_agent.go`

将模板数据中的 `AvailableTools` 改为分类展示：

```go
func (m *MainAgent) prepareTemplateData(ctx *AgentContext) map[string]any {
    data := make(map[string]any)

    // ... GameID, PlayerID, GameState 保持不变 ...

    // 只读工具信息
    readOnlyTools := m.registry.GetReadOnlyTools()
    toolInfo := make([]map[string]string, 0, len(readOnlyTools))
    for _, t := range readOnlyTools {
        toolInfo = append(toolInfo, map[string]string{
            "Name":        t.Name(),
            "Description": t.Description(),
        })
    }
    data["ReadOnlyTools"] = toolInfo
    data["AvailableTools"] = toolInfo  // 兼容旧模板

    return data
}
```

### Step 10: defaultSystemPrompt 同步更新

**文件**: `game_engine/agent/main_agent.go`

修改 `defaultSystemPrompt()` 只列出只读工具：

```go
parts = append(parts, "")
parts = append(parts, "可用Tools（只读查询）:")
for _, t := range m.registry.GetReadOnlyTools() {
    parts = append(parts, fmt.Sprintf("- `%s`: %s", t.Name(), t.Description()))
}
```

## 变更文件清单

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `game_engine/tool/tool.go` | 修改 | Tool 接口添加 `ReadOnly()`，BaseTool/EngineTool 添加 readOnly 字段 |
| `game_engine/tool/registry.go` | 修改 | 新增 `GetReadOnlyTools()`、`GetReadOnlySchemas()`、`GetAgentsForTool()` |
| `game_engine/tool/character_tools.go` | 修改 | 所有 NewXxxTool 传入 readOnly 参数 |
| `game_engine/tool/combat_tools.go` | 修改 | 所有 NewXxxTool 传入 readOnly 参数 |
| `game_engine/tool/rules_tools.go` | 修改 | 所有 NewXxxTool 传入 readOnly 参数 |
| `game_engine/tool/delegate_task_tool.go` | 不变 | BaseTool 默认 ReadOnly()=false 即可 |
| `game_engine/agent/main_agent.go` | 修改 | Tools() 只返回只读+delegate_task；Execute() 只暴露只读 schema；parseResponse() 拦截写操作调用 |
| `game_engine/agents.go` | 修改 | 写操作工具从 agents 列表移除 MainAgentName |
| `game_engine/prompt/main_system.md` | 修改 | 更新可用能力描述，明确读写分离规则 |

## 验证要点

1. `go build ./...` 编译通过
2. MainAgent 的 `Tools()` 只返回只读工具 + delegate_task
3. LLM 收到的 function calling schema 中不包含写操作工具
4. 即使 LLM 错误调用了写操作工具，`parseResponse()` 能拦截并转为 delegate_task
5. SubAgent 仍然拥有完整的工具集（不受影响）
6. 只读工具直调走原有 ToolCall → Execute 路径，无额外 LLM 调用
