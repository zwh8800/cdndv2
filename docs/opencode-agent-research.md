# OpenCode Agent 机制调研报告

## 概述

OpenCode 是一个开源的 AI 编程代理(143k stars)，采用 TypeScript 构建。其 Agent 系统设计简洁但功能强大，与 cdndv2 的 Go 实现有显著差异。本报告分析 OpenCode 的 Agent/SubAgent 实现，为 cdndv2 提供借鉴。

---

## OpenCode Agent 架构

### 核心概念

OpenCode 将 Agent 定义为**配置对象**而非执行类：

```typescript
// packages/opencode/src/agent/agent.ts
export const AgentInfo = z.object({
  name: z.string(),
  description: z.string().optional(),
  mode: z.enum(["subagent", "primary", "all"]),
  hidden: z.boolean().optional(),
  temperature: z.number().optional(),
  color: z.string().optional(),
  permission: Permission.Ruleset,
  model: z.object({ modelID: z.string(), providerID: z.string() }).optional(),
  prompt: z.string().optional(),
  options: z.record(z.string(), z.any()),
  steps: z.number().optional(),
})
```

### Agent 类型

| 类型 | 描述 | 调用方式 |
|------|------|----------|
| **primary** | 主Agent，用户直接交互 | Tab键切换 |
| **subagent** | 子Agent，被主Agent调用 | @提及或Task工具 |

### 内置 Agent

```typescript
const builtInAgents = {
  build: { mode: "primary", permission: defaults },   // 全权限
  plan: { mode: "primary", permission: readOnly },    // 只读
  general: { mode: "subagent" },                       // 通用子Agent
  explore: { mode: "subagent" },                       // 探索子Agent
}
```

---

## SubAgent 调用机制

### 1. Task 工具调用 (主要方式)

OpenCode 使用 **Task 工具** 调用子Agent，而非伪装成Tool：

```typescript
// 子Agent通过 Task 工具被调用
{
  name: "task",
  description: "Launch a new agent to handle complex, multistep tasks autonomously",
  parameters: {
    type: "object",
    properties: {
      command: { type: "string" },
      description: { type: "string" },
      subagent_type: { type: "string" },  // 指定子Agent类型
      task_id: { type: "string" },        // 可选：恢复之前任务
    }
  }
}
```

### 2. @ 提及调用

用户可直接 @ 提及子Agent：

```
@explore find all React components in src/
@general help me refactor this function
```

### 3. 子Agent 会话隔离

```typescript
// 启动子Agent时创建独立会话
const childSession = await Session.create({
  agent: subAgentName,
  parent: parentSessionID,
})

// 允许在主会话和子会话间切换
session_child_first   // 进入第一个子会话
session_child_cycle   // 循环切换子会话
session_parent        // 返回父会话
```

---

## 执行流程

### Session 驱动模式

OpenCode 不在 Agent 中执行，而是由 **Session** 模块驱动：

```
用户输入 → SessionPrompt.prompt() 
       → SessionPrompt.loop() 
       → SessionProcessor.process() 
       → LLM.stream() 
       → Tool Execution 
       → Loop
```

### 关键模块

| 模块 | 职责 |
|------|------|
| `prompt.ts` (~580行) | 主循环入口，消息构建，Tool解析 |
| `processor.ts` (~220行) | 流式处理，Tool生命周期，Doom Loop检测 |
| `llm.ts` (~200行) | LLM调用，System Prompt构建 |
| `compaction.ts` (~220行) | 上下文压缩，溢出处理 |

### SubAgent 执行流程

```
MainAgent LLM 返回 Task 工具调用
    │
    ▼
Task 工具执行
    │
    ├─► 创建子会话 (Session.create)
    │
    ▼
子Agent 独立执行 loop()
    │
    ▼
子会话结果返回给主会话
    │
    ▼
MainAgent 继续处理
```

---

## 权限控制

### Permission 系统

```typescript
// permission.ts
type Ruleset = {
  [tool: string]: "allow" | "deny" | "ask" | { [command: string]: "allow" | "deny" | "ask" }
}
```

### Agent 权限配置

```json
{
  "agent": {
    "build": {
      "permission": {
        "edit": "allow",
        "bash": "allow"
      }
    },
    "plan": {
      "permission": {
        "edit": "deny",
        "bash": "ask"
      }
    },
    "explore": {
      "permission": {
        "edit": "deny",
        "bash": "deny"
      }
    }
  }
}
```

### Task 权限控制

```json
{
  "agent": {
    "orchestrator": {
      "permission": {
        "task": {
          "*": "deny",
          "code-reviewer": "ask"
        }
      }
    }
  }
}
```

---

## 与 cdndv2 对比

### 架构差异

| 方面 | cdndv2 (Go) | OpenCode (TS) |
|------|-------------|---------------|
| **Agent 定义** | 接口 + 实现类 | 纯配置对象 (Zod Schema) |
| **执行方式** | Agent.Execute() 方法 | Session 驱动，共享执行路径 |
| **SubAgent 调用** | ToolCall 伪装 | 专用 Task 工具 |
| **状态管理** | AgentContext Metadata | Session 会话隔离 |
| **权限控制** | 集成在 ToolRegistry | 独立的 Permission 系统 |

### 调用协议差异

**cdndv2 (问题)**:
```go
// SubAgent调用伪装成Tool调用
subAgentCalls := m.extractSubAgentCalls(resp.ToolCalls)
// 返回格式看起来像Tool，但实际是Agent调用
```

**OpenCode (规范)**:
```typescript
// 专门的Task工具
{
  tool_calls: [{
    name: "task",
    arguments: {
      command: "search code",
      subagent_type: "explore"  // 明确指定子Agent
    }
  }]
}
```

### 子会话管理差异

**cdndv2**:
- SubAgent 结果存入 AgentContext.Metadata
- 通过 PhaseThink 回到 MainAgent
- 无会话隔离，状态混合

**OpenCode**:
- 创建独立的 Session
- 父子会话可切换
- 完全的状态隔离

---

## 可借鉴的设计

### 1. Agent 作为配置对象

cdndv2 可以将 Agent 简化为配置定义，让多个 Agent 共享同一个执行路径：

```typescript
// OpenCode 风格
type AgentConfig = {
  name: string
  mode: "primary" | "subagent"
  description: string
  systemPrompt?: string
  tools?: string[]
  permission?: Permission
  steps?: number
}
```

**优势**: 减少代码重复，统一执行逻辑

### 2. 专用 Task 工具

不应将 SubAgent 调用伪装成 ToolCall，而是使用专门的调用机制：

```go
// 新增 TaskTool
type TaskTool struct {
  subAgents map[string]SubAgent
}

func (t *TaskTool) Execute(ctx context.Context, args map[string]any) (string, error) {
  subAgentName := args["subagent_type"].(string)
  intent := args["command"].(string)
  
  subAgent := t.subAgents[subAgentName]
  resp, _ := subAgent.Execute(ctx, &AgentRequest{UserInput: intent})
  
  return resp.Content, nil
}
```

### 3. 子会话隔离

OpenCode 的子会话设计值得借鉴：

```go
// 创建子会话
func (l *ReActLoop) createSubSession(parentCtx *AgentContext, subAgent SubAgent) *AgentContext {
  return &AgentContext{
    GameID:   parentCtx.GameID,
    PlayerID: parentCtx.PlayerID,
    Engine:   parentCtx.Engine,
    History:  make([]llm.Message, 0),  // 独立历史
    Parent:   parentCtx,              // 链接父会话
  }
}
```

### 4. 权限系统分离

将权限从 ToolRegistry 分离出来：

```go
type Permission struct {
  Edit    string `json:"edit"`    // "allow", "deny", "ask"
  Bash    string `json:"bash"`
  WebFetch string `json:"webfetch"`
}

func (p *Permission) CanExecute(tool string) bool {
  switch p[tool] {
  case "allow": return true
  case "deny": return false
  case "ask": return false  // 需要用户确认
  }
  return false
}
```

### 5. @ 提及语法支持

在用户输入中解析 @ 语法：

```
用户输入: "用 @character_agent 创建一个法师"
              │
              ▼
解析为 SubAgent 调用
{
  agent_name: "character_agent",
  intent: "创建一个法师"
}
```

---

## 总结

### OpenCode 的优势

1. **简洁**: Agent 是配置而非类，减少代码量
2. **清晰**: SubAgent 调用使用专用 Task 工具，语义明确
3. **隔离**: 子会话独立状态，避免污染
4. **灵活**: 权限系统支持细粒度控制

### 对 cdndv2 的建议

1. **移除伪装**: 将 SubAgent 调用从 ToolCall 中分离
2. **引入 Task 工具**: 专门处理 Agent 调用
3. **会话隔离**: 为 SubAgent 创建独立上下文
4. **简化 Agent 定义**: 考虑配置化方式
5. **权限分离**: 将权限从 Tool 系统中解耦

### 实施优先级

| 优先级 | 改进项 | 预期效果 |
|--------|--------|----------|
| **高** | 移除 ToolCall 伪装 | 代码清晰度提升 |
| **高** | 引入 Task 工具 | 调用语义明确 |
| **中** | 子会话状态隔离 | 避免状态污染 |
| **中** | 权限系统分离 | 细粒度控制 |
| **低** | @ 语法支持 | 用户体验提升 |

---

## 参考资料

- [OpenCode 官方文档 - Agents](https://opencode.ai/docs/agents)
- [OpenCode Agent 源码](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/agent/agent.ts)
- [OpenCode Session 模块](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/session/)
- [Agent Source Code Analysis](https://www.opencode.live/source-code/agent/)