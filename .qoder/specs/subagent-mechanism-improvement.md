# SubAgent 机制改进方案

## Context

当前 cdndv2 的 SubAgent 机制存在多个设计缺陷：SubAgent 调用伪装成 ToolCall 导致语义不清和混合调用失败、串行执行导致延迟高、状态传递断裂导致 SubAgent 结果未被有效利用、MainAgent 职责过重同时承担路由和叙事、缺少结果合成步骤导致响应质量差。

本方案基于两份调研文档（`docs/subagent-improvement.md` 和 `docs/opencode-agent-research.md`），结合项目实际代码结构，采用三个核心设计决策：
- **Task Tool 方式**调用 SubAgent（兼容 OpenAI function calling）
- **独立 PhaseRoute** 分离路由职责
- **完整会话隔离**的 SubAgent 执行上下文

---

## 一、问题分析

### 1.1 SubAgent 调用伪装成 ToolCall

**代码位置**: [main_agent.go](game_engine/agent/main_agent.go:289) `extractSubAgentCalls()`

```go
// 当前实现：检查 ToolCall 名称是否匹配 SubAgent 名称
if _, ok := m.subAgents[call.Name]; ok {
    // 视为 SubAgent 调用
}
```

**具体问题**:
- LLM 将 SubAgent 视为普通 Tool，无法表达"委托给专家"的语义
- 混合 Tool+Agent 调用时逻辑断裂（[main_agent.go:321](game_engine/agent/main_agent.go:321)），遇到非 Agent ToolCall 直接返回 nil，丢失所有 Agent 调用
- MainAgent 的 SystemPrompt 将 Agent 和 Tool 并列展示，LLM 无法区分优先级

### 1.2 串行执行，无并行能力

**代码位置**: [react_loop.go](game_engine/react_loop.go:362) `executeSubAgents()`

```go
for _, call := range calls {  // 串行遍历
    resp, err := subAgent.Execute(ctx, req)
}
```

**具体问题**:
- 三个 SubAgent（character/combat/rules）只能依次执行
- 无 errgroup 或任何并发原语
- "攻击哥布林并检定陷阱"这类复合操作无法并行处理

### 1.3 状态传递断裂

**代码位置**: [react_loop.go](game_engine/react_loop.go:337) 结果存入 Metadata

```go
l.state.agentContext.Metadata["sub_agent_results"] = subAgentResults
// 注入历史的是通用字符串
l.state.History = append(l.state.History, llm.NewAssistantMessage(
    fmt.Sprintf("子Agent执行完成，共%d个结果", len(subAgentResults)), nil,
))
```

**具体问题**:
- `Metadata["sub_agent_results"]` 是 `map[string]*AgentResponse`，无类型安全保障
- MainAgent 收到的只是"共N个结果"这样的摘要，无法基于具体结果做叙事合成
- SubAgent 的 Tool 调用结果直接追加到主历史，污染上下文

### 1.4 缺少 Router 角色

**代码位置**: [main_agent.go](game_engine/agent/main_agent.go:87) `Execute()` 方法

MainAgent 的 `Execute()` 同时承担：意图理解、路由决策、Tool 调用、叙事生成。SystemPrompt 混合了路由指令和叙事指令。

### 1.5 缺少结果合成步骤

SubAgent 结果返回后直接进入 `PhaseThink`，MainAgent 没有专门的合成提示词，需要自己从历史中提取和整合结果，导致叙事质量不稳定。

---

## 二、设计目标

| 目标 | 衡量标准 |
|------|---------|
| 语义清晰的 Agent 调用 | LLM 能区分 Tool 调用和 Agent 委托 |
| 并行执行能力 | 独立 SubAgent 可并发执行，延迟降低 |
| 结构化状态传递 | AgentCallResult 类型安全，MainAgent 可精确引用结果 |
| 路由职责分离 | Router 只做路由，MainAgent 只做叙事 |
| 专门的结果合成 | 合成 Prompt 引导 MainAgent 基于结果生成叙事 |
| 会话隔离 | SubAgent 不污染主对话历史 |

---

## 三、技术方案

### 3.1 新增/修改类型定义

在 `game_engine/agent/agent.go` 中新增：

```go
// AgentCallResult Agent 执行结果（新增）
type AgentCallResult struct {
    AgentName string         `json:"agent_name"`
    Success   bool           `json:"success"`
    Content   string         `json:"content"`
    ToolCalls []llm.ToolCall `json:"tool_calls,omitempty"`
    State     map[string]any `json:"state,omitempty"`
    Error     string         `json:"error,omitempty"`
}

// ExecutionMode 执行模式（新增）
type ExecutionMode string

const (
    ExecutionSequential ExecutionMode = "sequential"
    ExecutionParallel   ExecutionMode = "parallel"
)

// RouterDecision 路由决策（新增）
type RouterDecision struct {
    TargetAgents  []AgentDelegation `json:"target_agents"`
    ExecutionMode ExecutionMode     `json:"execution_mode"`
    Reasoning     string            `json:"reasoning"`
    DirectResponse string           `json:"direct_response,omitempty"` // 无需Agent时的直接回复
}

// AgentDelegation 单个Agent委托（新增）
type AgentDelegation struct {
    AgentName string `json:"agent_name"`
    Intent    string `json:"intent"`
    Input     string `json:"input,omitempty"`
}
```

修改 `AgentContext`：

```go
type AgentContext struct {
    GameID       string
    PlayerID     string
    Engine       *engine.Engine
    History      []llm.Message
    CurrentState *game_summary.GameSummary
    Metadata     map[string]any

    // 新增
    AgentResults map[string]*AgentCallResult // SubAgent 执行结果
    Parent       *AgentContext               // 父会话引用（SubAgent隔离用）
    IsSubSession bool                        // 是否为子会话
}
```

修改 `NextAction` 枚举：

```go
const (
    ActionContinue        NextAction = iota
    ActionDelegate                         // 委托给SubAgent（替代原ActionCallSubAgent）
    ActionSynthesize                       // 合成结果（新增）
    ActionRespondToPlayer
    ActionWaitForInput
    ActionEndGame
)
```

### 3.2 Task Tool 方案：delegate_task 工具

新增文件 `game_engine/tool/delegate_task_tool.go`：

```go
// DelegateTaskTool 专门用于调用SubAgent的工具
// LLM通过标准function calling调用此工具，ReAct Loop拦截并路由到SubAgent
type DelegateTaskTool struct {
    engine any // 避免循环引用
}

func (t *DelegateTaskTool) Name() string { return "delegate_task" }
func (t *DelegateTaskTool) Description() string {
    return "将任务委托给专门的Agent处理。用于角色管理、战斗操作、规则检定等专业任务。"
}
func (t *DelegateTaskTool) ParametersSchema() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "agent_name": map[string]any{
                "type":        "string",
                "enum":        []string{"character_agent", "combat_agent", "rules_agent"},
                "description": "要委托的Agent名称",
            },
            "intent": map[string]any{
                "type":        "string",
                "description": "传递给Agent的任务意图描述",
            },
            "input": map[string]any{
                "type":        "string",
                "description": "额外的具体输入信息",
            },
        },
        "required":             []string{"agent_name", "intent"},
    }
}
func (t *DelegateTaskTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
    // 此Tool不会真正被ToolRegistry执行
    // ReAct Loop在PhaseAct中拦截delegate_task调用，路由到SubAgent
    return &ToolResult{
        Success: false,
        Error:   "delegate_task should be intercepted by ReActLoop, not executed by ToolRegistry",
    }, nil
}
```

**关键设计**：在 `react_loop.go` 的 `act()` 方法中，检测到 `delegate_task` 工具调用时，不走 `executeTools()` 路径，而是走 `executeDelegations()` 路径。

**混合调用处理**：当 MainAgent 同时返回 `delegate_task` 和普通 ToolCall 时：
1. 将 ToolCalls 分为两组：`delegateCalls`（name=="delegate_task"）和 `toolCalls`（其余）
2. 先执行普通 ToolCalls（同步），结果追加到历史
3. 再执行 delegate_tasks（可并行），结果存入 AgentContext.AgentResults
4. 全部完成后进入 PhaseSynthesize

### 3.3 RouterAgent 实现

新增文件 `game_engine/agent/router_agent.go`：

```go
type RouterAgent struct {
    llm    llm.LLMClient
    agents map[string]SubAgent
}

func (r *RouterAgent) Route(ctx context.Context, userInput string, history []llm.Message, gameState *game_summary.GameSummary) (*RouterDecision, error)
```

Router 不实现 `Agent` 接口（不是 Agent），而是作为 ReActLoop 的内部组件。它：
- 接收用户输入 + 游戏状态 + 对话历史
- 输出 `RouterDecision`（目标 Agent 列表 + 执行模式）
- 使用轻量 SystemPrompt，只做路由不做叙事
- 对于简单的纯叙事请求，`DirectResponse` 非空表示无需委托 Agent

### 3.4 SubAgent 会话隔离

在 `react_loop.go` 中新增：

```go
func (l *ReActLoop) createSubSession(parentCtx *AgentContext) *AgentContext {
    return &AgentContext{
        GameID:       parentCtx.GameID,
        PlayerID:     parentCtx.PlayerID,
        Engine:       parentCtx.Engine,
        History:      make([]llm.Message, 0), // 独立历史
        CurrentState: parentCtx.CurrentState,  // 共享游戏状态（只读）
        Metadata:     make(map[string]any),
        Parent:       parentCtx,               // 链接父会话
        IsSubSession: true,
    }
}
```

SubAgent 执行流程：
1. 创建子会话 `subCtx`
2. 注入 SubAgent 专属 SystemPrompt
3. 注入路由传来的 intent 作为 UserMessage
4. SubAgent 独立执行（可包含多轮 Tool 调用）
5. 返回 `AgentCallResult`，不修改父会话历史

### 3.5 并行执行

在 `react_loop.go` 中新增，使用 `golang.org/x/sync/errgroup`：

```go
func (l *ReActLoop) executeDelegationsParallel(ctx context.Context, delegations []AgentDelegation) map[string]*AgentCallResult {
    eg, ctx := errgroup.WithContext(ctx)
    mu := sync.Mutex{}
    results := make(map[string]*AgentCallResult, len(delegations))

    for _, d := range delegations {
        d := d
        eg.Go(func() error {
            result := l.executeSingleDelegation(ctx, d)
            mu.Lock()
            results[d.AgentName] = result
            mu.Unlock()
            return nil
        })
    }

    eg.Wait()
    return results
}
```

依赖分析决定执行模式：
- 无依赖的 Agent → 并行执行
- CombatAgent 依赖 CharacterAgent → 串行执行（先 Character 后 Combat）
- 同一 Agent 的多个委托 → 串行执行（状态冲突风险）

### 3.6 更新后的 ReAct Loop 阶段

```
PhaseObserve → PhaseRoute → PhaseThink → PhaseAct → PhaseSynthesize → PhaseWait/PhaseEnd
                 ↑                                          │
                 └──────────── (多轮迭代) ──────────────────┘
```

| 阶段 | 职责 | 新增/修改 |
|------|------|----------|
| PhaseObserve | 收集游戏状态 | 不变 |
| PhaseRoute | 路由决策 | **新增** |
| PhaseThink | MainAgent 思考 | 修改：移除路由职责，专注叙事 |
| PhaseAct | 执行工具/委托 | 修改：拦截 delegate_task |
| PhaseSynthesize | 合成结果 | **新增** |

**PhaseRoute 详情**：
1. RouterAgent.Route() 分析用户输入
2. 如果 `DirectResponse` 非空 → 跳转到 PhaseThink（纯叙事）
3. 如果有目标 Agent → 执行委托 → 跳转到 PhaseSynthesize
4. 如果意图不明确 → 跳转到 PhaseThink（让 MainAgent 询问玩家）

**PhaseSynthesize 详情**：
1. 检查 `AgentContext.AgentResults` 是否有结果
2. 构建 Synthesize Prompt（包含所有 AgentCallResult）
3. 调用 MainAgent.Synthesize() 生成叙事
4. 输出给玩家

### 3.7 MainAgent 修改

**移除**：
- `extractSubAgentCalls()` 方法（不再需要从 ToolCall 中提取 Agent 调用）
- `subAgents` 字段（路由职责移到 RouterAgent）

**新增**：
- `Synthesize(ctx, req) (*AgentResponse, error)` 方法
- 合成阶段专用 Prompt 模板

**修改**：
- `parseResponse()` 中 `ActionCallSubAgent` 改为 `ActionDelegate`
- SystemPrompt 更新：不再列出 SubAgent，改为介绍 `delegate_task` 工具
- `prepareTemplateData()` 新增 AgentResults 注入

---

## 四、实施步骤

### Phase 1: 规范化调用协议（优先级：高）

**目标**：移除 ToolCall 伪装，使用 delegate_task 工具

1. 新增 `game_engine/tool/delegate_task_tool.go`
2. 修改 `game_engine/agents.go`：注册 delegate_task 工具给 MainAgent
3. 修改 `game_engine/agent/main_agent.go`：
   - 移除 `extractSubAgentCalls()` 方法
   - 移除 `subAgents` 字段
   - 修改 `parseResponse()`：遇到 delegate_task 工具调用时设置 `ActionDelegate`
4. 修改 `game_engine/react_loop.go`：
   - 在 `act()` 中拦截 `delegate_task` 调用，路由到 `executeDelegations()`
   - `executeDelegations()` 替代原 `executeSubAgents()`
5. 修改 `game_engine/prompt/main_system.md`：
   - 移除"可用子Agent"段落
   - 新增 `delegate_task` 工具使用说明
6. 修改 `game_engine/agent/agent.go`：
   - `ActionCallSubAgent` 改为 `ActionDelegate`

**文件变更**：
- 新增: `game_engine/tool/delegate_task_tool.go`
- 修改: `game_engine/agent/agent.go`, `game_engine/agent/main_agent.go`, `game_engine/react_loop.go`, `game_engine/agents.go`, `game_engine/prompt/main_system.md`

### Phase 2: 状态传递增强（优先级：高）

**目标**：结构化 SubAgent 结果，类型安全

1. 修改 `game_engine/agent/agent.go`：
   - 新增 `AgentCallResult` 类型
   - 修改 `AgentContext`：新增 `AgentResults`, `Parent`, `IsSubSession`
   - 新增 `AddAgentResult()`, `GetAgentResult()` 方法
2. 修改 `game_engine/react_loop.go`：
   - `executeDelegations()` 返回 `map[string]*AgentCallResult`
   - 结果存入 `AgentContext.AgentResults`（不再用 Metadata）
   - 移除旧的历史注入逻辑
3. 实现会话隔离：
   - 新增 `createSubSession()` 方法
   - SubAgent 执行使用子会话
   - SubAgent 的 Tool 调用在子会话中执行

**文件变更**：
- 修改: `game_engine/agent/agent.go`, `game_engine/react_loop.go`

### Phase 3: 并行执行（优先级：中）

**目标**：独立 SubAgent 并发执行

1. 新增依赖：`go get golang.org/x/sync`
2. 修改 `game_engine/react_loop.go`：
   - 新增 `executeDelegationsParallel()` 方法
   - 新增 `executeSingleDelegation()` 方法
   - 新增依赖分析逻辑 `analyzeDependencies()`
   - 在 `act()` 中根据执行模式选择串行/并行
3. 新增 `game_engine/agent/agent.go`：
   - `ExecutionMode` 类型
   - `AgentDelegation` 类型

**文件变更**：
- 修改: `game_engine/react_loop.go`, `game_engine/agent/agent.go`, `go.mod`, `go.sum`

### Phase 4: Router 分离（优先级：中）

**目标**：独立路由决策

1. 新增 `game_engine/agent/router_agent.go`
2. 新增 `game_engine/prompt/router_system.md`
3. 修改 `game_engine/react_loop.go`：
   - 新增 `PhaseRoute` 阶段
   - 新增 `route()` 方法
   - 修改 `Run()` 主循环
4. 修改 `game_engine/engine.go`：创建 RouterAgent
5. 修改 `game_engine/agents.go`：集成 RouterAgent

**文件变更**：
- 新增: `game_engine/agent/router_agent.go`, `game_engine/prompt/router_system.md`
- 修改: `game_engine/react_loop.go`, `game_engine/engine.go`, `game_engine/agents.go`

### Phase 5: 结果合成（优先级：中）

**目标**：专门合成阶段

1. 新增 `game_engine/prompt/synthesize_system.md`
2. 修改 `game_engine/agent/main_agent.go`：
   - 新增 `Synthesize()` 方法
   - 新增 `buildSynthesizePrompt()` 方法
3. 修改 `game_engine/react_loop.go`：
   - 新增 `PhaseSynthesize` 阶段
   - 新增 `synthesize()` 方法
4. 修改 `game_engine/agent/agent.go`：
   - 新增 `ActionSynthesize`

**文件变更**：
- 新增: `game_engine/prompt/synthesize_system.md`
- 修改: `game_engine/agent/main_agent.go`, `game_engine/react_loop.go`, `game_engine/agent/agent.go`

---

## 五、预期收益

| 维度 | 改进前 | 改进后 |
|------|--------|--------|
| **语义清晰度** | SubAgent 伪装成 ToolCall，混合调用失败 | delegate_task 专用工具，语义明确 |
| **执行延迟** | 串行执行，3个Agent需3倍时间 | 并行执行，独立Agent同时运行 |
| **状态传递** | Metadata 泛型存储，结果仅为摘要 | AgentCallResult 类型安全，完整结果传递 |
| **职责分离** | MainAgent 承担路由+叙事 | Router专注路由，MainAgent专注叙事 |
| **响应质量** | 无合成步骤，叙事不稳定 | Synthesize阶段引导合成，叙事连贯 |
| **可维护性** | Agent调用逻辑散布在parseResponse中 | 调用/路由/合成各阶段独立 |
| **可扩展性** | 新增Agent需修改多处 | 新增Agent只需注册+配置 |

---

## 六、验证方案

### 6.1 编译验证
```bash
go build ./...
```
每个 Phase 完成后执行，确保编译通过。

### 6.2 单元测试（新增）

| 测试 | 验证内容 |
|------|---------|
| `TestDelegateTaskTool_Schema` | delegate_task 参数 schema 正确 |
| `TestExtractDelegations` | 从 ToolCall 中提取 AgentDelegation |
| `TestCreateSubSession` | 子会话隔离正确 |
| `TestAgentCallResult` | 结果传递完整 |
| `TestParallelExecution` | 并行执行正确且无竞态 |
| `TestRouterDecision` | 路由决策格式正确 |
| `TestDependencyAnalysis` | 依赖分析逻辑正确 |

### 6.3 集成测试

运行现有 `game_engine/engine_test.go` 中的 8 个集成测试，确保行为不退化：
```bash
OPENAI_API_KEY=sk-xxx go test ./game_engine/ -v -run TestGame
```

### 6.4 对比测试

使用相同输入对比改进前后的：
- 响应质量（叙事连贯性）
- 执行延迟（并行 vs 串行）
- 调试信息清晰度（delegate_task vs 伪装ToolCall）
