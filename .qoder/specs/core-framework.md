# D&D LLM 游戏引擎 - Phase 1 核心框架实现计划

## Context

本计划用于实现 D&D LLM 游戏引擎的 Phase 1 核心框架。项目采用多 Agent 协作 + ReAct 循环架构，让 LLM 扮演地下城主(DM)角色。核心原则：**game_engine 只负责调度，绝不自行运算任何游戏逻辑**，所有规则运算通过调用 dnd-core 引擎 API 完成。

项目依赖：
- 模块名: `github.com/zwh8800/cdndv2`
- Go 版本: 1.24.2
- dnd-core 引擎: `github.com/zwh8800/dnd-core` (本地路径)

---

## 实现内容

### 1. Tool Registry 基础框架

**文件: `game_engine/tool/tool.go`**
- `Tool` 接口: Name(), Description(), ParametersSchema(), Execute()
- `BaseTool` 基础实现: 封装 name, description, schema
- `EngineTool` 引擎 Tool 基类: 继承 BaseTool, 持有 *engine.Engine
- `ToolResult` 结构体: Success, Data, Message, Error, Metadata

**文件: `game_engine/tool/registry.go`**
- `ToolRegistry` 注册中心: tools map, byAgent map, category map
- 方法: Register(), Get(), GetByAgent(), GetByCategory(), GetAll()
- GetAll() 返回 LLM 函数调用格式的 Schema 列表

---

### 2. LLM 客户端接口

**文件: `game_engine/llm/message.go`**
- `MessageRole` 枚举: RoleSystem, RoleUser, RoleAssistant, RoleTool
- `Message` 结构体: Role, Content, ToolCalls, ToolResults, Name
- `ToolCall` 结构体: ID, Name, Arguments(map[string]any)
- `ToolResult` 结构体: ToolCallID, Content, IsError

**文件: `game_engine/llm/client.go`**
- `LLMClient` 接口: Complete(), Stream()
- `CompletionRequest`: Messages, Tools, Temperature, MaxTokens
- `CompletionResponse`: Content, ToolCalls, Usage, FinishReason
- `StreamChunk`: Delta, Done

**文件: `game_engine/llm/response.go`**
- ParseToolCalls(): 从原始响应解析 Tool 调用
- FormatToolResult(): 格式化 Tool 结果
- ExtractContent(): 提取纯文本内容

---

### 3. State 状态管理

**文件: `game_engine/state/summary.go`**
- `GameSummary` 结构体: 包装 dnd-core 的 StateSummary
- 扩展字段: PlayerInput, LastActionResult, AvailableActions
- `collectSummary()`: 收集游戏状态摘要

**文件: `game_engine/state/formatter.go`**
- FormatForLLM(): 格式化为 LLM 可读文本
- FormatCombatSummary(): 战斗状态格式化
- FormatActorSheet(): 角色信息格式化

---

### 4. Agent 系统

**文件: `game_engine/agent/agent.go`**
- `Agent` 接口: Name(), Description(), SystemPrompt(), Tools(), Execute()
- `SubAgent` 接口: 扩展 Agent, 添加 CanHandle(), Priority(), Dependencies()
- `AgentContext`: GameID, PlayerID, Engine, History, CurrentState, Metadata
- `AgentRequest`: UserInput, Intent, Context, SubAgentResults
- `AgentResponse`: Content, ToolCalls, NextAction, Errors
- `NextAction` 枚举: ActionContinue, ActionCallSubAgent, ActionRespondToPlayer, ActionWaitForInput, ActionEndGame

**文件: `game_engine/agent/main_agent.go`**
- `MainAgent` 结构体: registry, llm, subAgents, systemPromptTemplate
- Execute() 流程: 构建 System Prompt → 组装 Messages → 调用 LLM → 解析响应
- SystemPrompt(): 加载模板 + 注入状态摘要 + 添加 Tool/Agent 列表

**文件: `game_engine/context.go`**
- `Context` 结构体: 管理 AgentContext 的创建和更新
- 方法: NewContext(), UpdateHistory(), GetCurrentState()

---

### 5. Prompt 提示词管理

**文件: `game_engine/prompt/templates.go`**
- 使用 Go embed 嵌入 .md 文件
- LoadSystemPrompt(): 加载提示词
- RenderTemplate(): 渲染模板变量

**文件: `game_engine/prompt/main_system.md`**
- 角色定义: D&D 地下城主
- 核心原则: 规则至上、叙事驱动、玩家中心
- 可用能力: 子 Agent 列表
- 工作流程说明
- 输出格式要求
- 禁止行为

---

### 6. ReAct Loop 控制器

**文件: `game_engine/react_loop.go`**
- `ReActLoop` 结构体: engine, mainAgent, agents, tools, llm, state, maxIter
- `LoopState`: GameID, PlayerID, History, CurrentPhase, Iteration, PendingTools, LastResult
- `Phase` 枚举: PhaseObserve, PhaseThink, PhaseAct, PhaseWait, PhaseEnd
- `Run()`: 主循环入口
- `observe()`: 收集游戏状态, 构建上下文
- `think()`: 调用 mainAgent.Execute()
- `act()`: 处理 Tool 调用, 生成响应
- `executeTools()`: 执行 Tool 并返回结果
- `waitForInput()`: 等待玩家输入
- 错误处理: 可恢复错误继续循环, 不可恢复错误终止

---

### 7. 主引擎入口

**文件: `game_engine/engine.go`**
- `GameEngine` 结构体: dndEngine, reactLoop, registry, mainAgent, llmClient, config
- `NewGameEngine()`: 初始化所有组件
- `NewGame()`: 创建新游戏会话
- `LoadGame()`: 加载游戏存档
- `ProcessInput()`: 处理玩家输入
- `Close()`: 清理资源

---

## 目录结构

```
game_engine/
├── engine.go              # 主引擎入口
├── react_loop.go          # ReAct循环控制器
├── context.go             # 上下文管理
│
├── agent/
│   ├── agent.go           # Agent接口定义 + SubAgent接口
│   └── main_agent.go      # 主Agent(DM)实现
│
├── tool/
│   ├── tool.go            # Tool接口 + BaseTool + EngineTool + ToolResult
│   ├── registry.go        # ToolRegistry注册中心
│   ├── game_tools.go      # 游戏会话相关Tools (NewGame, LoadGame等)
│   └── actor_tools.go     # 角色相关Tools (GetActor, ListActors等)
│
├── llm/
│   ├── client.go          # LLMClient接口 + 请求/响应结构
│   ├── message.go         # Message + ToolCall + ToolResult + MessageRole
│   ├── response.go        # 响应解析工具函数
│   └── openai/
│       ├── client.go      # OpenAI客户端实现
│       └── config.go      # OpenAI配置
│
├── prompt/
│   ├── templates.go       # 提示词模板管理 + embed支持
│   └── main_system.md     # 主Agent系统提示词
│
└── state/
    ├── summary.go         # GameSummary + 状态收集
    └── formatter.go       # 状态格式化工具
```

---

## 实现顺序（按依赖关系）

### 第一批：基础接口（无依赖）
1. `tool/tool.go` - Tool 接口定义
2. `llm/message.go` - 消息格式定义

### 第二批：LLM 层
3. `llm/client.go` - LLMClient 接口
4. `llm/response.go` - 响应解析
5. `llm/openai/config.go` - OpenAI 配置
6. `llm/openai/client.go` - OpenAI 适配器

### 第三批：Tool 层
7. `tool/registry.go` - Tool 注册中心
8. `tool/game_tools.go` - 游戏 Tool 示例
9. `tool/actor_tools.go` - 角色 Tool 示例

### 第四批：State 层
10. `state/summary.go` - 状态摘要
11. `state/formatter.go` - 状态格式化

### 第五批：Agent 层
12. `agent/agent.go` - Agent 接口定义
13. `context.go` - 上下文管理
14. `prompt/templates.go` - 提示词模板
15. `prompt/main_system.md` - DM 系统提示词
16. `agent/main_agent.go` - 主 Agent 实现

### 第六批：核心引擎
17. `react_loop.go` - ReAct 循环控制器
18. `engine.go` - 主引擎入口

---

## 关键设计决策

1. **Tool 参数使用 `map[string]any`**: 直接映射 JSON，避免为每个 Tool 定义结构体
2. **EngineTool 持有引擎引用**: 简化 Tool 实现的参数传递
3. **LLM 客户端接口抽象**: 支持多种 LLM 提供商（OpenAI、Anthropic 等）
4. **Phase 1 不实现 Intent 解析**: 简化实现，让 Main Agent 直接处理输入
5. **复用 dnd-core 的 StateSummary**: 避免重复定义，通过包装扩展

---

## 验证方案

### 单元测试
- Tool 测试: Mock dnd-core Engine，测试参数解析和结果格式化
- Agent 测试: Mock LLMClient，测试 System Prompt 和响应解析
- ReAct Loop 测试: 测试状态转换、循环终止、错误处理

### 集成测试
- 使用 dnd-core 引擎真实 API
- 测试完整流程: 创建游戏 → 处理输入 → 输出响应

### 验证命令
```bash
# 运行所有测试
go test ./game_engine/...

# 运行特定包的测试
go test ./game_engine/tool/...
go test ./game_engine/agent/...
go test ./game_engine/...

# 编译检查
go build ./...
```

---

## 额外实现内容（根据用户确认）

### 8. OpenAI 客户端适配器

**文件: `game_engine/llm/openai/client.go`**
- `OpenAIClient` 结构体: 实现 LLMClient 接口
- 持有 OpenAI API client (使用官方 SDK `github.com/openai/openai-go`)
- `Complete()`: 调用 OpenAI Chat Completions API
- `Stream()`: 调用 OpenAI Streaming API
- 转换 Message 格式到 OpenAI 格式
- 解析 OpenAI 响应到 CompletionResponse

**文件: `game_engine/llm/openai/config.go`**
- `OpenAIConfig`: APIKey, Model, BaseURL (支持自定义端点)
- `DefaultOpenAIConfig()`: 默认配置

---

### 9. 基础 Tool 示例

**文件: `game_engine/tool/game_tools.go`**
- `NewGameTool`: 创建新游戏会话
  - Schema: name, setting
  - 调用 engine.NewGame()
- `LoadGameTool`: 加载游戏存档
- `SaveGameTool`: 保存游戏
- `ListGamesTool`: 列出所有存档

**文件: `game_engine/tool/actor_tools.go`**
- `GetActorTool`: 获取角色基本信息
  - Schema: game_id, actor_id
  - 调用 engine.GetActor()
- `ListActorsTool`: 列出所有角色
- `CreatePCTool`: 创建玩家角色（简化版，验证框架）

---

## 预期产出

Phase 1 完成后，将具备：
1. 完整的 Tool 框架，可注册和执行任意 Tool
2. Main Agent 基础实现，可调用 LLM 并解析响应
3. ReAct 循环控制器，可执行 observe-think-act 循环
4. LLM 客户端接口 + OpenAI 适配器，可直接调用 OpenAI API
5. 基础 Tool 示例（NewGame、GetActor 等），验证框架可用

后续 Phase 2 将在此框架上实现更多具体 Tool（战斗、检定等）和子 Agent。
