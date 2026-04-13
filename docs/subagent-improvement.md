# SubAgent 机制改进方案

## 背景

当前 cdndv2 的 SubAgent 机制存在多个设计问题，与主流框架(LangGraph、CrewAI、AutoGen)的实现有较大差距。本文档详细描述改进方案。

---

## 当前问题总结

| 问题 | 严重程度 | 影响 |
|-----|---------|------|
| SubAgent 调用伪装成 ToolCall | 高 | 语义不清，调试困难 |
| 串行执行，无并行能力 | 高 | 延迟高用户体验差 |
| 状态传递断裂 | 高 | SubAgent 结果未被有效利用 |
| 缺少 Router 角色 | 中 | MainAgent 职责过重 |
| 意图匹配过于简单 | 中 | 路由不准确 |
| 缺少结果合成步骤 | 高 | 响应质量差 |

---

## 改进目标

1. **规范化调用机制**: 使用显式的 Agent 调用格式
2. **支持并行执行**: 多 SubAgent 可同时执行
3. **完善状态管理**: SubAgent 结果正确传递给 MainAgent
4. **分离路由职责**: Router LLM 专门负责路由
5. **添加结果合成**: MainAgent 基于 SubAgent 结果合成最终响应

---

## 改进方案

### 1. 重新设计 Agent 调用协议

#### 1.1 移除 ToolCall 伪装

**当前(问题代码)**:
```go
// main_agent.go - 通过 ToolCall 触发 SubAgent
subAgentCalls := m.extractSubAgentCalls(resp.ToolCalls)
```

**改进方案**: 使用结构化的 Agent 调用格式，让 LLM 直接返回 SubAgent 调用。

**新增类型**:

```go
// AgentCall - 显式的 Agent 调用
type AgentCall struct {
    AgentName string `json:"agent_name"` // 子 Agent 名称
    Intent    string `json:"intent"`    // 传递给子 Agent 的意图描述
    Input     string `json:"input"`     // 额外输入
    Context   map[string]any `json:"context"` // 传递的上下文
}

// AgentCallResult - Agent 执行结果
type AgentCallResult struct {
    AgentName string         `json:"agent_name"`
    Success   bool           `json:"success"`
    Content   string        `json:"content"`
    ToolCalls []ToolCall     `json:"tool_calls"`  // Agent 产生的 Tool 调用
    State     map[string]any `json:"state"`     // Agent 执行后的状态变更
    Error     string        `json:"error,omitempty"`
}

// MainAgent 响应增强
type MainAgentResponse struct {
    Content     string        `json:"content"`       // 叙事内容
    AgentCalls  []AgentCall   `json:"agent_calls"`   // Agent 调用请求
    ToolCalls   []ToolCall    `json:"tool_calls"`    // 直接 Tool 调用
    NextAction  NextAction    `json:"next_action"`   // 下一步动作
    StateChange *StateChange  `json:"state_change"`  // 状态变更
}

// NextAction 枚举更新
const (
    ActionContinue       NextAction = iota // 继续思考
    ActionCallAgents                       // 调用 Agent (新)
    ActionParallelAgents                  // 并行调用 Agents (新)
    ActionSynthesize                       // 合成结果 (新)
    ActionRespondToPlayer
    ActionWaitForInput
    ActionEndGame
)
```

#### 1.2 Tool 定义为 Agent 调用

**移除伪装**:
- 不再使用 `character_agent` 作为 Tool name
- 让 LLM 知道这些是 Agent 而非普通 Tool

**在 system prompt 中明确声明**:

```
## 可用 Agents

你可以通过调用以下专门的 Agent 来处理特定任务：

- **character_agent**: 角色管理专家
  - 职责: 角色创建、查询、更新、经验、升级、休息
  - 调用方式: 返回 agent_call 格式

- **combat_agent**: 战斗管理专家
  - 职责: 战斗初始化、回合管理、攻击、伤害、治疗
  - 调用方式: 返回 agent_call 格式

- **rules_agent**: 规则仲裁专家
  - 职责: 检定、豁免、法术、专注管理
  - 调用方式: 返回 agent_call 格式

## Agent 调用格式

当你需要调用 Agent 时，请按以下格式返回：

{
  "agent_calls": [
    {
      "agent_name": "character_agent",
      "intent": "创建1级人类法师",
      "input": "角色名: 张三, 职业: 法师, 种族: 人类",
      "context": {}
    }
  ]
}

## 重要规则

1. 所有游戏规则执行必须通过 Tool 调用 dnd-core API 完成
2. 不要尝试自行计算规则结果，让专门的 Agent 处理
3. Agent 调用是异步的，你只需要指定调用，不需要等待结果
4. 收到 Agent 结果后，你可以基于结果生成最终响应
```

---

### 2. 实现并行执行

#### 2.1 React Loop 修改

**当前(串行)**:
```go
// react_loop.go
func (l *ReActLoop) executeSubAgents(ctx context.Context, calls []AgentCall) map[string]*AgentCallResult {
    results := make(map[string]*AgentCallResult)
    for _, call := range calls {  // 串行执行
        resp, err := subAgent.Execute(ctx, req)
        // ...
    }
    return results
}
```

**改进(并行)**:
```go
// 并行执行 Agent 调用
func (l *ReActLoop) executeAgentsParallel(ctx context.Context, calls []AgentCall) map[string]*AgentCallResult {
    if len(calls) == 0 {
        return nil
    }

    // 使用 errgroup 实现并行
    var mu sync.Mutex
    results := make(map[string]*AgentCallResult, len(calls))
    
    eg, ctx := errgroup.WithContext(ctx)
    
    for _, call := range calls {
        call := call  // 闭包捕获
        eg.Go(func() error {
            result := l.executeSingleAgent(ctx, call)
            mu.Lock()
            results[call.AgentName] = result
            mu.Unlock()
            return nil
        })
    }
    
    if err := eg.Wait(); err != nil {
        // 处理错误，可能部分成功
    }
    
    return results
}

// 单独执行一个 Agent
func (l *ReActLoop) executeSingleAgent(ctx context.Context, call AgentCall) *AgentCallResult {
    subAgent, ok := l.agents[call.AgentName]
    if !ok {
        return &AgentCallResult{
            AgentName: call.AgentName,
            Success:   false,
            Error:     "agent not found: " + call.AgentName,
        }
    }
    
    req := &AgentRequest{
        UserInput: call.Intent,
        Input:     call.Input,
        Intent:    call.Intent,
        Context:   call.Context,
    }
    
    resp, err := subAgent.Execute(ctx, req)
    if err != nil {
        return &AgentCallResult{
            AgentName: call.AgentName,
            Success:   false,
            Error:     err.Error(),
        }
    }
    
    // 执行 Agent 产生的 Tool 调用
    var toolResults []ToolResult
    if len(resp.ToolCalls) > 0 {
        toolResults = l.executeTools(ctx, resp.ToolCalls)
    }
    
    return &AgentCallResult{
        AgentName: call.AgentName,
        Success:   true,
        Content:   resp.Content,
        ToolCalls: resp.ToolCalls,
        State:     resp.StateChange,
    }
}
```

#### 2.2 并行结果合并

```go
// 并行执行后的结果合并
func (l *ReActLoop) synthesizeAgentResults(results map[string]*AgentCallResult) string {
    var parts []string
    
    for agentName, result := range results {
        if !result.Success {
            parts = append(parts, fmt.Sprintf("[%s] 失败: %s", agentName, result.Error))
            continue
        }
        
        summary := fmt.Sprintf("[%s] 执行完成: %s", agentName, result.Content)
        if len(result.ToolCalls) > 0 {
            summary += fmt.Sprintf(" (执行了 %d 个工具)", len(result.ToolCalls))
        }
        parts = append(parts, summary)
    }
    
    return strings.Join(parts, "\n")
}
```

---

### 3. 完善状态传递

#### 3.1 AgentContext 增强

```go
type AgentContext struct {
    GameID       string
    PlayerID     string
    Engine       *engine.Engine
    History      []llm.Message
    CurrentState *game_summary.GameSummary
    Metadata     map[string]any
    
    // 新增：Agent 执行结果链
    AgentResults map[string]*AgentCallResult  // 所有 Agent 的执行结果
    AgentChain   []AgentCall                   // 调用链记录
}

// AddAgentResult 添加 Agent 执行结果
func (c *AgentContext) AddAgentResult(result *AgentCallResult) {
    if c.AgentResults == nil {
        c.AgentResults = make(map[string]*AgentCallResult)
    }
    c.AgentResults[result.AgentName] = result
}

// GetAgentResult 获取特定 Agent 的结果
func (c *AgentContext) GetAgentResult(agentName string) *AgentCallResult {
    return c.AgentResults[agentName]
}

// GetAllAgentResults 获取所有 Agent 结果
func (c *AgentContext) GetAllAgentResults() map[string]*AgentCallResult {
    return c.AgentResults
}
```

#### 3.2 SubAgent 结果注入 MainAgent 上下文

```go
// react_loop.go - 在重新调用 MainAgent 前注入结果
func (l *ReActLoop) handleAgentCalls(ctx context.Context, result *MainAgentResponse) Phase {
    // 执行 Agent 调用
    agentResults := l.executeAgentsParallel(ctx, result.AgentCalls)
    
    // 注入到上下文
    for _, res := range agentResults {
        l.state.agentContext.AddAgentResult(res)
    }
    
    // 生成结果摘要给 MainAgent
    resultsSummary := l.synthesizeAgentResults(agentResults)
    
    // 添加到历史
    l.state.History = append(l.state.History, llm.NewAssistantMessage(
        fmt.Sprintf("Agent 执行结果:\n%s", resultsSummary),
        nil,
    ))
    
    // 返回 Think 阶段，让 MainAgent 合成最终响应
    return PhaseThink
}
```

#### 3.3 MainAgent SystemPrompt 包含 Agent 结果

```go
func (m *MainAgent) prepareTemplateData(ctx *AgentContext) map[string]any {
    data := make(map[string]any)
    // ... 现有字段 ...
    
    // 新增：Agent 执行结果
    if len(ctx.AgentResults) > 0 {
        var agentResults []map[string]any
        for name, result := range ctx.AgentResults {
            agentResults = append(agentResults, map[string]any{
                "name":    name,
                "success": result.Success,
                "content": result.Content,
                "error":   result.Error,
            })
        }
        data["AgentResults"] = agentResults
    }
    
    return data
}
```

---

### 4. 分离 Router 角色

#### 4.1 引入 Router Agent

```go
type RouterAgent struct {
    llm    llm.LLMClient
    agents map[string]SubAgent
}

// Router 职责：只做路由决策，不执行具体任务
type RouterDecision struct {
    TargetAgents []string   `json:"target_agents"`  // 目标 Agent 列表
    Intent       string     `json:"intent"`          // 传递给 Agent 的意图
    ExecutionMode string   `json:"execution_mode"` // "sequential" | "parallel"
    Reasoning    string    `json:"reasoning"`       // 路由理由
}

// NewRouterAgent 创建 Router Agent
func NewRouterAgent(llmClient llm.LLMClient, agents map[string]SubAgent) *RouterAgent {
    return &RouterAgent{
        llm:    llmClient,
        agents: agents,
    }
}

// Route 路由决策
func (r *RouterAgent) Route(ctx context.Context, userInput string, history []llm.Message) (*RouterDecision, error) {
    systemPrompt := r.buildRouterPrompt()
    
    messages := []llm.Message{
        llm.NewSystemMessage(systemPrompt),
    }
    messages = append(messages, history...)
    messages = append(messages, llm.NewUserMessage(userInput))
    
    resp, err := r.llm.Complete(ctx, &llm.CompletionRequest{
        Messages: messages,
    })
    if err != nil {
        return nil, err
    }
    
    // 解析路由决策
    return r.parseRouterResponse(resp.Content)
}
```

#### 4.2 Router Prompt 模板

```markdown
你是一个专业的任务路由专家。你的任务是根据用户输入，决定应该调用哪些专业 Agent。

## 你的职责

1. 分析用户输入的意图
2. 选择最合适的 Agent 来处理
3. 决定是串行还是并行执行

## 可用 Agent

{{range .Agents}}
- **{{.Name}}**: {{.Description}}
  - 优先级: {{.Priority}}
  - 依赖: {{.Dependencies}}
{{end}}

## 决策指南

1. **单一任务**: 如果用户请求只涉及一个领域，直接路由到对应的 Agent
2. **多任务**: 如果请求涉及多个领域，可以并行路由到多个 Agent
3. **依赖任务**: 如果一个 Agent 依赖另一个，先执行依赖方
4. **不确定**: 如果不确定，询问用户澄清

## 输出格式

请以 JSON 格式输出你的决策：

{
  "target_agents": ["character_agent", "rules_agent"],
  "intent": "玩家想要创建角色并进行一次感知检定",
  "execution_mode": "parallel",
  "reasoning": "这两个任务是独立的，可以并行执行"
}

注意：
- target_agents 必须是有效的 Agent 名称
- intent 应该清晰描述需要 Agent 完成的任务
- execution_mode 可以是 "sequential" 或 "parallel"
```

#### 4.3 执行流程改造

```
玩家输入
    │
    ▼
┌─────────────────────────────────────────┐
│  Phase: Route (路由阶段) - 新增          │
│  - RouterAgent.Route()                  │
│  - 分析意图，选择目标 Agent              │
│  - 决定执行模式                          │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│  Phase: Execute (执行阶段)              │
│  - 并行/串行执行目标 Agents              │
│  - 收集执行结果                          │
│  - 注入到上下文                          │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│  Phase: Synthesize (合成阶段) - 新增    │
│  - MainAgent 基于 Agent 结果生成响应     │
│  - 包含叙事内容                          │
└─────────────────────────────────────────┘
    │
    ▼
输出给玩家
```

---

### 5. 添加结果合成

#### 5.1 MainAgent 合成阶段

```go
// MainAgent 添加 Synthesize 方法
func (m *MainAgent) Synthesize(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
    // 基于 Agent 结果生成最终响应
    systemPrompt := m.buildSynthesizePrompt(req.Context)
    
    messages := []llm.Message{
        llm.NewSystemMessage(systemPrompt),
    }
    messages = append(messages, req.Context.History...)
    
    // 强制要求生成最终响应
    messages = append(messages, llm.NewUserMessage(
        "请基于上述 Agent 执行结果，生成最终响应给玩家。",
    ))
    
    resp, err := m.llm.Complete(ctx, &llm.CompletionRequest{
        Messages: messages,
    })
    if err != nil {
        return nil, err
    }
    
    return &AgentResponse{
        Content:    resp.Content,
        NextAction: ActionRespondToPlayer,
    }, nil
}

// 构建合成阶段的 System Prompt
func (m *MainAgent) buildSynthesizePrompt(ctx *AgentContext) string {
    var parts []string
    
    parts = append(parts, "你是 Dungeon Master。你刚刚调用了专业的 Agent 来处理玩家的请求，现在需要基于结果生成最终响应。")
    parts = append(parts, "")
    parts = append(parts, "## Agent 执行结果")
    
    for agentName, result := range ctx.AgentResults {
        parts = append(parts, fmt.Sprintf("### %s", agentName))
        if result.Success {
            parts = append(parts, result.Content)
        } else {
            parts = append(parts, fmt.Sprintf("错误: %s", result.Error))
        }
        parts = append(parts, "")
    }
    
    parts = append(parts, "")
    parts = append(parts, "## 你的任务")
    parts = append(parts, "基于上述结果，用引人入胜的方式描述给玩家。")
    parts = append(parts, "包括：")
    parts = append(parts, "- 发生了什么")
    parts = append(parts, "- 结果如何")
    parts = append(parts, "- 接下来需要做什么(如果有)")
    
    return strings.Join(parts, "\n")
}
```

#### 5.2 React Loop 集成

```go
// react_loop.go - PhaseSynthesize 处理
func (l *ReActLoop) handleSynthesize(ctx context.Context, result *MainAgentResponse) Phase {
    // 如果有 Agent 结果，需要合成
    if len(l.state.agentContext.AgentResults) > 0 {
        synthesizeReq := &agent.AgentRequest{
            Context: l.state.agentContext,
        }
        
        // 调用 MainAgent 的 Synthesize 方法
        synthesized, err := l.mainAgent.Synthesize(ctx, synthesizeReq)
        if err != nil {
            // 降级到直接输出 Agent 结果
            return PhaseWait
        }
        
        // 记录合成的响应
        l.state.LastResponse = synthesized.Content
        
        // 添加到历史
        l.state.History = append(l.state.History, llm.NewAssistantMessage(
            synthesized.Content,
            nil,
        ))
        
        return PhaseWait
    }
    
    return PhaseWait
}
```

---

## 改进后的完整流程

```
玩家输入: "我攻击哥布林，并尝试寻找隐藏的陷阱"
    │
    ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Phase: Observe                                                      │
│ - CollectSummary() 获取游戏状态                                      │
└─────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Phase: Route (新增)                                                 │
│ - RouterAgent 分析意图                                               │
│ - 决策: parallel (combat_agent + rules_agent)                       │
│ - Reasoning: "攻击和感知检定可以并行执行"                              │
└─────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Phase: Execute                                                      │
│ - 并行执行 combat_agent 和 rules_agent                               │
│ - combat_agent: 处理攻击哥布林                                       │
│ - rules_agent: 处理感知检定                                          │
│ - 收集结果，注入到 AgentContext                                      │
└─────────────────────────────────────────────────────────────────────┘
    │
    ├─► combat_agent 结果: "攻击造成 8 点伤害"
    ├─► rules_agent 结果: "感知检定 DC 15，投掷 18，成功"
    │
    ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Phase: Think (再次)                                                 │
│ - MainAgent 收到 AgentResults                                       │
│ - SystemPrompt 包含所有 Agent 结果                                  │
│ - 生成最终响应                                                       │
└─────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Phase: Synthesize (新增)                                            │
│ - "你挥剑砍向哥布林，剑锋切入血肉，造成 8 点伤害！"                   │
│ - "同时，你仔细搜寻四周，在墙边发现了一个隐藏的陷阱触发装置。"         │
└─────────────────────────────────────────────────────────────────────┘
    │
    ▼
玩家看到完整响应
```

---

## 实现优先级

| 阶段 | 任务 | 优先级 | 预计工作量 |
|-----|------|--------|-----------|
| Phase 1 | 规范化调用协议 | 高 | 1周 |
| Phase 2 | 状态传递增强 | 高 | 1周 |
| Phase 3 | 并行执行 | 中 | 1周 |
| Phase 4 | Router 分离 | 中 | 1周 |
| Phase 5 | 结果合成 | 中 | 1周 |

---

## 文件变更清单

### 新增文件

1. `game_engine/agent/router_agent.go` - Router Agent 实现
2. `game_engine/prompt/router_system.md` - Router System Prompt 模板
3. `game_engine/prompt/synthesize_system.md` - 合成阶段 System Prompt

### 修改文件

1. `game_engine/agent/agent.go` - 新增类型定义
2. `game_engine/agent/main_agent.go` - 新增 Synthesize 方法
3. `game_engine/react_loop.go` - 新增 PhaseRoute, PhaseSynthesize
4. `game_engine/prompt/main_system.md` - 更新 Agent 调用说明
5. `docs/agent-flowchart.md` - 更新流程图

---

## 测试计划

1. **单元测试**: 
   - RouterAgent 路由决策测试
   - 并行执行结果正确性测试
   - 状态传递完整性测试

2. **集成测试**:
   - 完整流程测试(输入→路由→执行→合成→输出)
   - 并行执行性能测试
   - 错误处理测试

3. **对比测试**:
   - 改进前后响应质量对比
   - 延迟对比(预期降低 50%)

---

## 风险与缓解

| 风险 | 缓解措施 |
|-----|---------|
| LLM 不按格式返回 Agent 调用 | 增强 prompt + 后备解析 |
| 并行执行导致资源竞争 | 使用 Go 协程 + mutex |
| Router 决策错误 | 保留 MainAgent 覆盖能力 |
| 响应质量下降 | A/B 测试对比 |