# Game Engine Agent 流程图

## 概述

本文件介绍 cdndv2 中 game_engine/agent 包的多Agent系统架构与执行流程。

## 架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              游戏会话层                                      │
│                         (GameEngine / ReactLoop)                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          MainAgent (DM)                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  职责: 意图理解 │ 任务分解 │ 叙事生成 │ 玩家交互 │ Tool/SubAgent调用  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│  - SystemPrompt: 从模板加载动态生成                                          │
│  - Tools: ~92个 D&D Engine APIs                                             │
│  - SubAgents: CharacterAgent, CombatAgent, RulesAgent                     │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                 ▼
        ┌───────────────────┐ ┌───────────────────┐ ┌───────────────────┐
        │ CharacterAgent    │ │  CombatAgent      │ │  RulesAgent       │
        │ (角色管理)        │ │  (战斗管理)       │ │  (规则仲裁)       │
        ├───────────────────┤ ├───────────────────┤ ├───────────────────┤
        │ 创建/查询角色     │ │ 战斗初始化       │ │ 能力检定         │
        │ 经验/升级        │ │ 回合管理         │ │ 技能检定         │
        │ 休息(短休/长休)  │ │ 攻击/伤害        │ │ 豁免检定         │
        │                   │ │ 治疗/移动        │ │ 法术施放         │
        │ Priority: 10     │ │ Priority: 20     │ │ 专注管理         │
        │                   │ │                   │ │ Priority: 5      │
        │ Deps: -           │ │ Deps: Character   │ │ Deps: -           │
        └───────────────────┘ └───────────────────┘ └───────────────────┘
                    │                 │                 │
                    └─────────────────┼─────────────────┘
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ToolRegistry                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  ~92个 D&D Engine APIs                                               │   │
│  │  - Character Tools: create_actor, get_actor, add_experience...     │   │
│  │  - Combat Tools: start_combat, execute_attack, deal_damage...        │   │
│  │  - Rules Tools: perform_skill_check, cast_spell, concentration...  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          D&D Core Engine                                    │
│         (github.com/zwh8800/dnd-core)                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  所有规则计算、伤害计算、状态管理都在dnd-core中执行                   │   │
│  │  game_engine 永远不进行任何游戏逻辑计算                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 核心接口

### Agent 接口

```go
type Agent interface {
    Name() string
    Description() string
    SystemPrompt(ctx *AgentContext) string
    Tools() []tool.Tool
    Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
}
```

### SubAgent 接口

```go
type SubAgent interface {
    Agent
    CanHandle(intent string) bool      // 判断能否处理该意图
    Priority() int                    // 处理优先级
    Dependencies() []string            // 依赖的其他Agent
}
```

### AgentContext 上下文

```go
type AgentContext struct {
    GameID       string
    PlayerID     string
    Engine       *engine.Engine       // D&D引擎实例
    History      []llm.Message       // 对话历史
    CurrentState *game_summary.GameSummary
    Metadata     map[string]any
}
```

### AgentResponse 响应

```go
type AgentResponse struct {
    Content       string
    ToolCalls     []llm.ToolCall
    SubAgentCalls []SubAgentCall
    NextAction    NextAction          // ActionContinue, ActionCallSubAgent, ActionRespondToPlayer, ActionWaitForInput, ActionEndGame
    StateChange   *StateChange
    Errors        []AgentError
}
```

## 执行流程

### 整体流程 (ReAct Loop)

```
玩家输入
    │
    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Phase: Observe (观察阶段)                                                │
│  - 调用 CollectSummary() 获取游戏状态                                     │
│  - 更新 AgentContext                                                      │
└────────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Phase: Think (思考阶段)                                                  │
│  - MainAgent.Execute()                                                     │
│  - 构建 SystemPrompt (从模板 + 游戏状态 + Tools + SubAgents)              │
│  - 构建 Messages (system + history + user_input)                         │
│  - 调用 LLM.Complete()                                                    │
│  - 解析 LLM 响应                                                          │
└────────────────────────────────────────────────────────────────────────────┘
    │
    ├──► 有ToolCalls?
    │     │
    │     ▼ (是)
    │  ┌──────────────────────────────────────────────────────────────────┐ │
    │  │ Phase: Act (行动阶段) - Tool调用                                  │ │
    │  │ 1. 检查是否为SubAgent调用                                       │ │
    │  │    - 是: 提取SubAgentCalls, 设置NextAction=ActionCallSubAgent   │ │
    │  │    - 否: 执行Tool调用, 调用D&D Engine API                       │ │
    │  │ 2. 获取Tool执行结果                                             │ │
    │  │ 3. 将结果添加到历史                                           │ │
    │  │ 4. 回到Think阶段继��                                          │ │
    │  └──────────────────────────────────────────────────────────────────┘ │
    │     │
    │     └──► 继续循环 (max 10次)
    │
    ▼ (无ToolCalls)
┌────────────────────────────────────────────────────────────────────────────┐
│  Phase: Act (行动阶段) - 响应玩家                                           │
│  - 有Content?                                                              │
│    - 是: 设置NextAction=ActionRespondToPlayer, 输出给玩家                     │
│    - 否: 设置NextAction=ActionWaitForInput, 等待玩家输入                    │
└────────────────────────────────────────────────────────────────────────────┘
    │
    ▼
玩家看到响应 / 等待输入
```

### MainAgent.Execute 详细流程

```
MainAgent.Execute(userInput)
    │
    ▼
1. 构建 SystemPrompt
   - LoadSystemPrompt("main_system.md")
   - prepareTemplateData(ctx): 游戏ID, 玩家ID, 游戏状态, Tools列表, SubAgents列表
   - RenderTemplate(template, data)
    │
    ▼
2. 构建 Messages
   - [System] systemPrompt
   - [History...] req.Context.History
   - [User] req.UserInput (如果不在history中)
    │
    ▼
3. 获取 Tools 定义
   - registry.GetAll() → []Tool (92个)
    │
    ▼
4. 调用 LLM
   - llm.Complete(Messages, Tools)
    │
    ▼
5. 解析 LLM 响应
   - parseResponse(resp)
   - 检查 ToolCalls
   - 提取 SubAgentCalls / ToolCalls
   - 确定 NextAction
    │
    ▼
返回 AgentResponse
    │
    ├──► SubAgentCalls → 调用子Agent
    ├──► ToolCalls → 执行Tool后继续
    └──► Content → 响应玩家
```

### SubAgent 调用流程

```
MainAgent 返回 SubAgentCalls
    │
    ▼
对于每个 SubAgentCall:
    │
    ▼
1. 获取对应的 SubAgent
   - subAgents[call.AgentName]
    │
    ▼
2. 构建 SubAgent 请求
   - AgentRequest{
       UserInput: call.Intent,
       Intent: call.Intent,
       Context: req.Context,
       SubAgentResults: 当前已执行的子Agent结果
     }
    │
    ▼
3. 执行 SubAgent
   - subAgent.Execute(ctx, subReq)
    │
    ▼
4. 获取 SubAgent 响应
   - agentResp.ToolCalls → 执行Tool调用
   - agentResp.Content → 追加到结果
    │
    ▼
5. 将结果存入 SubAgentResults
   - 回到 MainAgent 继续处理
```

## Agent 详情

### MainAgent (主Agent/DM)

- **名称**: `main_agent`
- **描述**: 主Agent(DM)，负责意图理解、任务分解、叙事生成、玩家交互
- **优先级**: N/A (主Agent)
- **依赖**: 所有SubAgent
- **职责**:
  - 接收玩家输入
  - 理解玩家意图
  - 决定调用哪些SubAgent或Tool
  - 生成叙事响应
  - 控制游戏流程

### CharacterAgent (角色管理)

- **名称**: `character_agent`
- **描述**: 角色管理Agent，负责角色创建、查询、更新、经验、休息
- **优先级**: 10
- **依赖**: 无
- **可处理意图**:
  - `create_character`, `create_pc`, `create_npc`, `create_enemy`, `create_companion`
  - `get_actor`, `get_pc`, `list_actors`, `update_actor`, `remove_actor`
  - `add_experience`, `level_up`, `short_rest`, `long_rest`
  - 关键词: character, 角色, 创建, 升级, 经验, 休息

### CombatAgent (战斗管理)

- **名称**: `combat_agent`
- **描述**: 战斗管理Agent，负责战斗初始化、回合管理、攻击、伤害治疗
- **优先级**: 20
- **依赖**: CharacterAgent
- **可处理意图**:
  - `attack`, `combat`, `turn`, `damage`, `heal`, `move`
  - `start_combat`, `end_combat`, `next_turn`, `execute_attack`
  - 关键词: 战斗, 攻击, 回合, 伤害, 治疗, 移动

### RulesAgent (规则仲裁)

- **名称**: `rules_agent`
- **描述**: 规则仲裁Agent，负责检定、豁免、法术施放、专注管理
- **优先级**: 5
- **依赖**: 无
- **可处理意图**:
  - `check`, `save`, `spell`, `concentration`, `skill`
  - `perform_ability_check`, `perform_skill_check`, `perform_saving_throw`
  - `cast_spell`, `get_spell_slots`, `concentration_check`
  - 关键词: 检定, 豁免, 法术, 专注, 技能

## 流程示例

### 示例1: 玩家攻击敌人

```
玩家输入: "我攻击哥布林!"
    │
    ▼
MainAgent.Execute("我攻击哥布林!")
    │
    ▼
[Think] LLM决定需要调用combat_agent
    │
    ▼
[Act] MainAgent返回SubAgentCalls: [combat_agent]
    │
    ▼
CombatAgent.Execute("攻击哥布林")
    │
    ▼
[Think] LLM调用 execute_attack Tool
    │
    ▼
[Act] Tool调用 dnd-core API
    │
    ▼
返回 ToolResult (伤害值: 8)
    │
    ▼
MainAgent生成响应: "你对哥布林造成了8点伤害!"
    │
    ▼
输出给玩家
```

### 示例2: 玩家创建角色

```
玩家输入: "创建一个名为张三的法师"
    │
    ▼
MainAgent.Execute("创建一个名为张三的法师")
    │
    ▼
[Think] LLM决定需要调用character_agent
    │
    ▼
[Act] MainAgent返回SubAgentCalls: [character_agent]
    │
    ▼
CharacterAgent.Execute("创建角色张三, 职业法师")
    │
    ▼
[Think] LLM调用 create_actor Tool
    │
    ▼
[Act] Tool调用 dnd-core API
    │
    ▼
返回 ToolResult (角色创建成功)
    │
    ▼
MainAgent生成响应: "角色张三, 1级法师, 已创建完成!"
    │
    ▼
输出给玩家
```

### 示例3: 玩家进行技能检定

```
玩家输入: "我尝试搜索陷阱"
    │
    ▼
MainAgent.Execute("我尝试搜索陷阱")
    │
    ▼
[Think] LLM决定需要调用rules_agent
    │
    ▼
[Act] MainAgent返回SubAgentCalls: [rules_agent]
    │
    ▼
RulesAgent.Execute("搜索陷阱, 感知检定")
    │
    ▼
[Think] LLM调用 perform_skill_check Tool
    │
    ▼
[Act] Tool调用 dnd-core API
    │
    ▼
返回 ToolResult (DC 15, 投掷 18, 成功)
    │
    ▼
MainAgent生成响应: "你仔细搜索后发现了一个隐藏的陷阱!"
    │
    ▼
输出给玩家
```

## 状态机

### NextAction 状态

| 状态 | 值 | 说明 |
|------|-----|------|
| ActionContinue | 0 | 继续思考, 有ToolCall待执行 |
| ActionCallSubAgent | 1 | 调用子Agent |
| ActionRespondToPlayer | 2 | 生成响应给玩家 |
| ActionWaitForInput | 3 | 等待玩家输入 |
| ActionEndGame | 4 | 结束游戏 |

### ReAct Loop 状态机

```
┌──────────┐    输入     ┌─────────┐
│  Observe │ ───────► │  Think  │
└──────────┘          └─────────┘
                           │
                           ▼ LLM响应
                    ┌──────────────┐
                    │              │
              有ToolCalls      无ToolCalls
                    │              │
                    ▼              ▼
              ┌───────────┐  ┌──────────────┐
              │   Act     │  │ Respond     │
              │(Tool/Sub) │  │  toPlayer  │
              └───────────┘  └──────────────┘
                    │
                    ▼ 循环(max 10)
              ┌───────────┐
              │  Wait     │ ◄──────┐
              └───────────┘       │
                                │ 玩家输入
                                 └────────
```

## 设计原则

1. **永远不自行计算**: 所有游戏规则执行必须通过Tool调用dnd-core API完成
2. **主从架构**: MainAgent是唯一入口, SubAgent是分工 specialist
3. **意图分发**: MainAgent根据意图调用合适的SubAgent
4. **状态传递**: 通过AgentContext在Agent间传递游戏状态
5. **循环保护**: ReAct Loop最多循环10次防止无限循环

## 文件结构

```
game_engine/agent/
├── agent.go           # 核心接口定义 (Agent, SubAgent, Context, Request, Response)
├── const.go          # 常量定义 (Agent名称)
├── main_agent.go    # MainAgent实现 (DM)
├── character_agent.go  # CharacterAgent实现
├── combat_agent.go     # CombatAgent实现
└── rules_agent.go     # RulesAgent实现
```