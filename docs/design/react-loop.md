# ReAct循环设计

## 0. 核心原则

> **game_engine 绝不自行运算任何游戏逻辑。**
> ReAct循环中的Act阶段，所有游戏操作必须通过Tool调用dnd引擎。
> LLM负责决策判断（做什么），引擎负责规则运算（怎么做），game_engine只负责调度。

## 1. 概述

ReAct (Reasoning + Acting) 循环是游戏引擎的核心调度机制，负责协调LLM推理和D&D引擎交互。每个循环周期包含观察(Observe)、思考(Think)、行动(Act)三个阶段。

## 2. 循环流程

### 2.1 整体流程图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         ReAct Loop Controller                            │
└─────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │   START      │
                              └──────┬───────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Phase 1: OBSERVE (观察)                                                 │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ • 收集游戏状态 (场景、角色、战斗状态等)                           │    │
│  │ • 获取玩家输入                                                   │    │
│  │ • 生成状态摘要                                                   │    │
│  │ • 更新对话历史                                                   │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Phase 2: THINK (思考)                                                   │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ • 调用主Agent(DM)                                                │    │
│  │ • 主Agent分析意图                                                │    │
│  │ • 决定是否需要调用子Agent                                        │    │
│  │ • 决定是否需要执行Tool                                           │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
                         ┌───────────────────────┐
                         │   需要调用子Agent?    │
                         └───────────┬───────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
                    ▼                ▼                ▼
               ┌────────┐      ┌────────┐      ┌────────┐
               │  YES   │      │  NO    │      │ Tool   │
               │ 子Agent│      │ 生成   │      │ 调用   │
               └────┬───┘      │ 响应   │      └────┬───┘
                    │          └────┬───┘           │
                    │               │               │
                    ▼               │               ▼
┌─────────────────────────┐        │    ┌─────────────────────────┐
│  执行子Agent             │        │    │  执行Tool               │
│  • 调用子Agent           │        │    │  • ToolRegistry.Get     │
│  • 子Agent调用Tools      │        │    │  • Tool.Execute         │
│  • 返回结果              │        │    │  • 返回ToolResult       │
└─────────────────────────┘        │    └─────────────────────────┘
                    │               │               │
                    └───────────────┼───────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Phase 3: ACT (行动)                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ • 处理执行结果                                                   │    │
│  │ • 更新游戏状态                                                   │    │
│  │ • 生成玩家响应                                                   │    │
│  │ • 检查是否需要继续思考                                           │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
                         ┌───────────────────────┐
                         │   继续循环?           │
                         └───────────┬───────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
                    ▼                ▼                ▼
               ┌────────┐      ┌────────┐      ┌────────┐
               │ 继续   │      │ 等待   │      │ 结束   │
               │ OBSERVE│      │ 输入   │      │ 游戏   │
               └────┬───┘      └────┬───┘      └────┬───┘
                    │               │               │
                    └───────────────┼───────────────┘
                                    │
                              ┌─────┴─────┐
                              │   LOOP    │
                              └───────────┘
```

### 2.2 游戏Phase与循环状态的关系

ReAct循环是Agent的决策循环，而引擎Phase是D&D游戏世界的状态。两者关系如下：

```
┌─────────────────────────────────────────────────────────────────────┐
│                        ReAct循环状态                                 │
│  PhaseObserve → PhaseThink → PhaseAct → (循环或等待输入)             │
│                                                                      │
│  这是LLM Agent的推理周期，每轮循环包含完整的观察-思考-行动过程         │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              │ 每个循环内
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        引擎Phase状态                                 │
│  PhaseCharacterCreation → PhaseExploration → PhaseCombat/PhaseRest   │
│                                                                      │
│  这是D&D游戏世界的状态，决定哪些操作被引擎允许                         │
│  - PhaseCharacterCreation: 角色创建/装备/准备                         │
│  - PhaseExploration: 探索/交互/任务/开始战斗                         │
│  - PhaseCombat: 回合制战斗                                           │
│  - PhaseRest: 短休/长休恢复                                          │
└─────────────────────────────────────────────────────────────────────┘
```

每个ReAct循环周期内：
1. **Observe**阶段收集当前引擎Phase
2. **Think**阶段Agent根据Phase决定调用哪些Tools
3. **Act**阶段执行符合Phase的操作（引擎会拒绝不符合的操作）

## 3. 核心组件

### 3.1 ReActLoop 控制器

```go
// ReActLoop ReAct循环控制器
type ReActLoop struct {
    engine     *engine.Engine
    mainAgent  Agent
    agents     map[string]SubAgent
    tools      *ToolRegistry
    llm        LLMClient
    state      *LoopState
    maxIter    int
}

// LoopState 循环状态
type LoopState struct {
    GameID       model.ID
    PlayerID     model.ID
    History      []Message
    CurrentPhase Phase
    Iteration    int
    PendingTools []ToolCall
    LastResult   *AgentResponse
}

// Phase 循环阶段
type Phase int

const (
    PhaseObserve Phase = iota
    PhaseThink
    PhaseAct
    PhaseWait
    PhaseEnd
)
```

### 3.2 循环执行

```go
// Run 执行ReAct循环
func (l *ReActLoop) Run(ctx context.Context, initialInput string) error {
    // 初始化
    l.state.CurrentPhase = PhaseObserve
    l.state.Iteration = 0

    for l.state.CurrentPhase != PhaseEnd {
        if l.state.Iteration >= l.maxIter {
            return fmt.Errorf("max iterations reached")
        }

        switch l.state.CurrentPhase {
        case PhaseObserve:
            l.observe(ctx)

        case PhaseThink:
            response, err := l.think(ctx)
            if err != nil {
                return err
            }
            l.state.LastResult = response
            l.state.CurrentPhase = PhaseAct

        case PhaseAct:
            nextPhase := l.act(ctx)
            l.state.CurrentPhase = nextPhase

        case PhaseWait:
            // 等待玩家输入
            input := l.waitForInput()
            l.state.History = append(l.state.History, Message{
                Role:    "user",
                Content: input,
            })
            l.state.CurrentPhase = PhaseObserve
        }

        l.state.Iteration++
    }

    return nil
}

// observe 观察阶段
func (l *ReActLoop) observe(ctx context.Context) {
    // 1. 收集游戏状态
    stateSummary := l.collectStateSummary(ctx)

    // 2. 构建上下文
    agentCtx := &AgentContext{
        GameID:       l.state.GameID,
        PlayerID:     l.state.PlayerID,
        Engine:       l.engine,
        History:      l.state.History,
        CurrentState: stateSummary,
    }

    // 3. 存储上下文
    l.agentContext = agentCtx

    // 4. 转入思考阶段
    l.state.CurrentPhase = PhaseThink
}

// think 思考阶段
func (l *ReActLoop) think(ctx context.Context) (*AgentResponse, error) {
    // 调用主Agent
    req := &AgentRequest{
        Context: l.agentContext,
    }

    if len(l.state.History) > 0 {
        lastMsg := l.state.History[len(l.state.History)-1]
        if lastMsg.Role == "user" {
            req.UserInput = lastMsg.Content
        }
    }

    return l.mainAgent.Execute(ctx, req)
}

// act 行动阶段
func (l *ReActLoop) act(ctx context.Context) Phase {
    result := l.state.LastResult

    // 1. 处理Tool调用
    if len(result.ToolCalls) > 0 {
        toolResults := l.executeTools(ctx, result.ToolCalls)
        // 将结果添加到历史
        l.state.History = append(l.state.History, Message{
            Role:        "tool",
            ToolCalls:   result.ToolCalls,
            ToolResults: toolResults,
        })
        return PhaseThink // 继续思考
    }

    // 2. 处理子Agent调用
    if result.NextAction == ActionCallSubAgent {
        // 子Agent调用会在Tool调用中处理
        return PhaseThink
    }

    // 3. 生成响应给玩家
    if result.Content != "" {
        l.state.History = append(l.state.History, Message{
            Role:    "assistant",
            Content: result.Content,
        })
        // 输出给玩家
        l.outputToPlayer(result.Content)
    }

    // 4. 根据下一步动作决定
    switch result.NextAction {
    case ActionWaitForInput:
        return PhaseWait
    case ActionEndGame:
        return PhaseEnd
    default:
        return PhaseWait
    }
}
```

### 3.3 状态摘要生成

```go
// StateSummary 状态摘要
type StateSummary struct {
    // 游戏基本信息
    GameID   model.ID `json:"game_id"`
    GameName string   `json:"game_name"`
    GameTime string   `json:"game_time"`

    // 当前场景
    CurrentScene *SceneSummary `json:"current_scene"`

    // 玩家角色
    Player *ActorSummary `json:"player"`

    // 战斗状态
    Combat *CombatSummary `json:"combat,omitempty"`

    // 任务状态
    ActiveQuests []QuestSummary `json:"active_quests,omitempty"`

    // 最近的NPC
    NearbyNPCs []ActorSummary `json:"nearby_npcs,omitempty"`
}

// collectStateSummary 收集状态摘要
func (l *ReActLoop) collectStateSummary(ctx context.Context) *StateSummary {
    summary := &StateSummary{
        GameID: l.state.GameID,
    }

    // 获取当前场景
    sceneResult, _ := l.engine.GetCurrentScene(ctx, engine.GetCurrentSceneRequest{
        GameID: l.state.GameID,
    })
    if sceneResult != nil {
        summary.CurrentScene = l.summarizeScene(sceneResult)
    }

    // 获取玩家角色
    pcResult, _ := l.engine.GetPC(ctx, engine.GetPCRequest{
        GameID: l.state.GameID,
        PCID:   l.state.PlayerID,
    })
    if pcResult != nil {
        summary.Player = l.summarizeActor(pcResult.PC)
    }

    // 获取战斗状态
    combatResult, _ := l.engine.GetCurrentCombat(ctx, engine.GetCurrentCombatRequest{
        GameID: l.state.GameID,
    })
    if combatResult != nil && combatResult.Combat.Status == model.CombatActive {
        summary.Combat = l.summarizeCombat(combatResult)
    }

    // 获取活跃任务
    questsResult, _ := l.engine.GetActorQuests(ctx, engine.GetActorQuestsRequest{
        GameID:  l.state.GameID,
        ActorID: l.state.PlayerID,
    })
    for _, quest := range questsResult.Quests {
        if quest.Status == model.QuestActive {
            summary.ActiveQuests = append(summary.ActiveQuests, l.summarizeQuest(quest))
        }
    }

    return summary
}
```

## 4. D&D引擎Phase与游戏阶段处理

### 4.1 引擎Phase定义

dnd-core引擎定义了4个Phase，控制不同阶段允许执行的操作：

| Phase | 说明 | 主要操作 |
|-------|------|----------|
| PhaseCharacterCreation | 角色创建阶段 | 创建/更新/删除角色、检定、库存管理、准备法术 |
| PhaseExploration | 探索阶段（默认） | 所有常规操作、创建场景、任务管理、开始战斗 |
| PhaseCombat | 战斗阶段 | 攻击、移动、法术、回合管理 |
| PhaseRest | 休息阶段 | 短休/长休、HP恢复、法术位恢复 |

### 4.2 Phase状态机

```
┌─────────────────────────────────────────────────────────────────┐
│                    D&D引擎Phase状态机                            │
└─────────────────────────────────────────────────────────────────┘

     ┌────────────────────┐
     │ CharacterCreation  │ 游戏初始阶段
     │  (角色创建)        │
     └─────────┬──────────┘
               │ SetPhase(Exploration)
               │ "角色创建完成，冒险开始"
               ▼
     ┌────────────────────┐
     │   Exploration      │ 默认阶段
     │   (探索)           │◀────────────────────────┐
     └─────────┬──────────┘                         │
               │ SetPhase(Combat)                   │
               │ "战斗开始"                         │
               ▼                                    │
     ┌────────────────────┐                         │
     │      Combat        │                        │
     │    (战斗)          │                         │
     └─────────┬──────────┘                         │
               │ SetPhase(Exploration)              │
               │ "战斗结束"                         │
               ▼                                    │
     ┌────────────────────┐                         │
     │   Exploration ◀────┼────────────────────────┘
     │   (探索)           │
     └─────────┬──────────┘
               │ SetPhase(Rest)
               │ "开始长休"
               ▼
     ┌────────────────────┐
     │      Rest          │
     │    (休息)          │
     └─────────┬──────────┘
               │ SetPhase(Exploration)
               │ "长休完成"
               ▼
     ┌────────────────────┐
     │   Exploration      │
     │   (探索)           │
     └────────────────────┘
```

### 4.3 Phase与ReAct循环的关系

**重要**: ReAct循环是LLM Agent的决策循环，而引擎Phase是D&D游戏世界的状态。两者是不同层次的概念：

```
┌─────────────────────────────────────────────────────────────────┐
│                    ReAct循环 (LLM层)                             │
│                                                                 │
│   OBSERVE ──▶ THINK ──▶ ACT ──▶ (循环)                          │
│                                                                 │
│   每个ReAct循环周期内，Agent可以：                               │
│   - 查询当前Phase                                               │
│   - 根据Phase决定可调用哪些Tools                                │
│   - 执行符合当前Phase的操作                                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Agent调用Tools
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    引擎Phase (D&D世界层)                         │
│                                                                 │
│   Phase决定：                                                   │
│   - 哪些操作被允许                                              │
│   - 哪些操作被拒绝 (ErrPhaseNotAllowed)                         │
│                                                                 │
│   示例: Combat阶段不允许 create_pc                              │
│        Exploration阶段不允许 execute_attack                     │
└─────────────────────────────────────────────────────────────────┘
```

### 4.4 Phase感知Tool调用

ReAct循环在执行Tool时，必须考虑当前Phase：

```go
// executeToolWithPhaseCheck 带Phase检查的Tool执行
func (l *ReActLoop) executeToolWithPhaseCheck(ctx context.Context, call ToolCall) (*ToolResult, error) {
    // 1. 获取当前Phase
    currentPhase, err := l.engine.GetPhase(ctx, l.state.GameID)
    if err != nil {
        return &ToolResult{Success: false, Error: err.Error()}, nil
    }

    // 2. 获取允许的操作列表
    allowedOps, err := l.engine.GetAllowedOperations(ctx, l.state.GameID)
    if err != nil {
        return &ToolResult{Success: false, Error: err.Error()}, nil
    }

    // 3. 检查操作是否被允许
    op := l.toolToOperation(call.Name)
    if !isOperationAllowed(op, allowedOps) {
        return &ToolResult{
            Success: false,
            Error:   fmt.Sprintf("当前阶段(%s)不允许执行此操作", currentPhase),
            Message: fmt.Sprintf("当前处于%s阶段，无法执行%s。请先切换到合适的阶段。", currentPhase, call.Name),
        }, nil
    }

    // 4. 执行Tool
    return l.executeTool(ctx, call)
}
```

### 4.5 各Phase的典型处理流程

#### PhaseCharacterCreation (角色创建阶段)

```
角色创建阶段处理流程:

1. 引导玩家创建角色
   - 询问角色名称、种族、职业、属性
   - 调用 create_pc Tool

2. 装备和法术准备
   - 添加初始装备 (add_item, equip_item)
   - 准备法术 (prepare_spells)

3. 创建初始场景
   - 调用 create_scene Tool

4. 切换到探索阶段
   - 调用 SetPhase(Exploration)
   - 冒险正式开始
```

#### PhaseExploration (探索阶段)

```go
// handleExplorationPhase 处理探索阶段
func (l *ReActLoop) handleExplorationPhase(ctx context.Context, input string) {
    // 解析玩家意图
    intent := l.parseIntent(input)

    switch intent.Type {
    case IntentMove:
        // 移动到新位置或场景
        l.handleMovement(ctx, intent)

    case IntentInteract:
        // 与场景物品或NPC交互
        l.handleInteraction(ctx, intent)

    case IntentCheck:
        // 进行检定
        l.handleCheck(ctx, intent)

    case IntentRest:
        // 切换到休息阶段
        l.engine.SetPhase(ctx, l.state.GameID, model.PhaseRest, "玩家开始长休")

    case IntentStartCombat:
        // 切换到战斗阶段
        l.engine.SetPhase(ctx, l.state.GameID, model.PhaseCombat, "遭遇敌人")
        l.handleCombat(ctx, input)

    case IntentCreatePC:
        // 创建新角色（探索阶段也允许）
        l.handleCreatePC(ctx, intent)

    default:
        // 默认叙事处理
        l.handleNarrative(ctx, intent)
    }
}
```

#### PhaseCombat (战斗阶段)

```go
// handleCombatPhase 处理战斗阶段
func (l *ReActLoop) handleCombatPhase(ctx context.Context, input string) {
    // 检查是否玩家回合
    turnResult, _ := l.engine.GetCurrentTurn(ctx, engine.GetCurrentTurnRequest{
        GameID: l.state.GameID,
    })

    if turnResult != nil && turnResult.ActorID != l.state.PlayerID {
        // 不是玩家回合，处理敌人行动
        l.handleEnemyTurn(ctx)
        return
    }

    // 解析战斗意图
    intent := l.parseCombatIntent(input)

    switch intent.Type {
    case IntentAttack:
        l.executePlayerAttack(ctx, intent)

    case IntentCastSpell:
        l.executePlayerSpell(ctx, intent)

    case IntentMove:
        l.executePlayerMovement(ctx, intent)

    case IntentAction:
        l.executePlayerAction(ctx, intent)

    case IntentEndTurn:
        l.engine.NextTurn(ctx, engine.NextTurnRequest{
            GameID: l.state.GameID,
        })

    case IntentEndCombat:
        l.engine.SetPhase(ctx, l.state.GameID, model.PhaseExploration, "战斗结束")
    }
}
```

#### PhaseRest (休息阶段)

```go
// handleRestPhase 处理休息阶段
func (l *ReActLoop) handleRestPhase(ctx context.Context, input string) {
    intent := l.parseRestIntent(input)

    switch intent.Type {
    case IntentShortRest:
        // 短休
        result, _ := l.engine.ShortRest(ctx, engine.ShortRestRequest{
            GameID:  l.state.GameID,
            ActorID: l.state.PlayerID,
        })
        l.outputToPlayer(formatRestResult(result))

    case IntentLongRest:
        // 长休 - 需要8小时
        l.engine.SetPhase(ctx, l.state.GameID, model.PhaseExploration, "长休完成")
        // 应用长休效果
        result, _ := l.engine.EndLongRest(ctx, engine.EndLongRestRequest{
            GameID:  l.state.GameID,
            ActorID: l.state.PlayerID,
        })
        l.outputToPlayer(formatLongRestResult(result))

    case IntentEndRest:
        // 提前结束休息
        l.engine.SetPhase(ctx, l.state.GameID, model.PhaseExploration, "休息提前结束")
    }
}
```

### 4.6 Phase切换示例

```
[Phase: OBSERVE]
当前Phase: Exploration
玩家输入: "我们找个安全的地方休息吧"

[Phase: THINK]
主Agent推理:
1. 玩家想要休息
2. 需要切换到Rest阶段
3. 调用 SetPhase(Rest)

[Phase: ACT]
Tool调用:
- set_phase(phase: "rest", reason: "队伍决定长休")

引擎响应:
{
    "old_phase": "exploration",
    "new_phase": "rest",
    "message": "队伍开始长休，需要8小时才能完成"
}

[Phase: THINK]
主Agent生成响应:
"你们在附近的山洞里找到了一个安全的庇护所。
点燃营火后，你们准备开始长休。
8小时的休息将恢复你们的生命值和法术位。

在休息期间，你们想要：
1. 开始短休（1小时，恢复部分HP）
2. 开始长休（8小时，完全恢复）
3. 保持警戒休息"

[Phase: WAIT]
等待玩家输入...
```

### 4.7 Phase错误处理

当Agent尝试执行当前Phase不允许的操作时：

```
[Phase: Exploration]
玩家输入: "我要攻击那个商人"

[Agent决策]
Agent尝试调用: execute_attack

[引擎拒绝]
Error: ErrPhaseNotAllowed
Message: "当前阶段(exploration)不允许执行execute_attack"

[错误处理]
1. 将错误返回给Agent
2. Agent理解当前不是战斗阶段
3. Agent调整策略:
   - 如果玩家真要战斗: 先调用start_combat
   - 如果玩家是口误: 重新理解意图

[Agent响应]
"你突然想要攻击商人？这意味着战斗即将开始！
如果你确定要这么做，我需要先开始一场战斗。
你要对商人发起攻击吗？这将导致严重的后果..."
```

## 5. 消息格式

### 5.1 消息定义

```go
// Message 对话消息
type Message struct {
    Role        string       `json:"role"`         // user, assistant, tool, system
    Content     string       `json:"content"`      // 消息内容
    ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`  // Tool调用请求
    ToolResults []ToolResult `json:"tool_results,omitempty"` // Tool调用结果
    Name        string       `json:"name,omitempty"`  // Tool名称(用于tool角色)
}

// ToolCall Tool调用请求
type ToolCall struct {
    ID        string         `json:"id"`
    Name      string         `json:"name"`
    Arguments map[string]any `json:"arguments"`
}

// ToolResult Tool调用结果
type ToolResult struct {
    ToolCallID string `json:"tool_call_id"`
    Content    string `json:"content"`
    IsError    bool   `json:"is_error"`
}
```

### 5.2 对话历史管理

```go
// HistoryManager 对话历史管理器
type HistoryManager struct {
    maxMessages int
    messages    []Message
}

// Add 添加消息
func (h *HistoryManager) Add(msg Message) {
    h.messages = append(h.messages, msg)

    // 如果超出限制，进行压缩
    if len(h.messages) > h.maxMessages {
        h.compress()
    }
}

// compress 压缩历史消息
func (h *HistoryManager) compress() {
    // 保留系统消息和最近的消息
    var systemMsgs []Message
    var recentMsgs []Message

    for _, msg := range h.messages {
        if msg.Role == "system" {
            systemMsgs = append(systemMsgs, msg)
        }
    }

    // 保留最近的消息
    startIdx := len(h.messages) - h.maxMessages/2
    if startIdx > 0 {
        recentMsgs = h.messages[startIdx:]
    }

    // 合并
    h.messages = append(systemMsgs, recentMsgs...)
}

// ToLLMMessages 转换为LLM消息格式
func (h *HistoryManager) ToLLMMessages() []LLMMessage {
    var llmMsgs []LLMMessage
    for _, msg := range h.messages {
        llmMsg := LLMMessage{
            Role:    msg.Role,
            Content: msg.Content,
        }
        if len(msg.ToolCalls) > 0 {
            llmMsg.ToolCalls = msg.ToolCalls
        }
        if msg.Role == "tool" {
            llmMsg.Name = msg.Name
        }
        llmMsgs = append(llmMsgs, llmMsg)
    }
    return llmMsgs
}
```

## 6. 错误处理

### 6.1 错误类型

```go
// LoopError 循环错误
type LoopError struct {
    Type    ErrorType
    Message string
    Cause   error
}

type ErrorType int

const (
    ErrorEngine      ErrorType = iota // 引擎错误
    ErrorLLM                          // LLM调用错误
    ErrorTool                         // Tool执行错误
    ErrorValidation                   // 验证错误
    ErrorTimeout                      // 超时错误
    ErrorMaxIterations                // 最大迭代错误
)
```

### 6.2 错误处理策略

```go
// handleError 处理错误
func (l *ReActLoop) handleError(err *LoopError) Phase {
    switch err.Type {
    case ErrorEngine:
        // 引擎错误，尝试恢复或告知玩家
        if isRecoverable(err.Cause) {
            l.outputToPlayer("发生了一个错误，让我们重试...")
            return PhaseThink
        }
        l.outputToPlayer("抱歉，游戏遇到了问题：" + err.Message)
        return PhaseWait

    case ErrorLLM:
        // LLM错误，可能需要重试
        l.outputToPlayer("让我再想想...")
        return PhaseThink

    case ErrorTool:
        // Tool执行错误，告知LLM调整
        // 将错误信息返回给Agent重新思考
        l.state.History = append(l.state.History, Message{
            Role:    "tool",
            Content: fmt.Sprintf("Error: %s", err.Message),
        })
        return PhaseThink

    case ErrorMaxIterations:
        // 最大迭代，需要输出当前结果
        l.outputToPlayer("抱歉，我遇到了一些困难，请提供更多信息。")
        return PhaseWait

    default:
        return PhaseWait
    }
}
```

### 6.3 恢复机制

```go
// recoverableErrors 可恢复的引擎错误
var recoverableErrors = []error{
    engine.ErrNotYourTurn,
    engine.ErrActionAlreadyUsed,
    engine.ErrOutOfRange,
}

func isRecoverable(err error) bool {
    for _, e := range recoverableErrors {
        if errors.Is(err, e) {
            return true
        }
    }
    return false
}
```

## 7. 性能优化

### 7.1 状态缓存

```go
// StateCache 状态缓存
type StateCache struct {
    scene    *SceneSummary
    player   *ActorSummary
    combat   *CombatSummary
    expireAt time.Time
    ttl      time.Duration
}

// GetScene 获取缓存的场景
func (c *StateCache) GetScene() *SceneSummary {
    if time.Now().Before(c.expireAt) {
        return c.scene
    }
    return nil
}

// Update 更新缓存
func (c *StateCache) Update(scene *SceneSummary, player *ActorSummary, combat *CombatSummary) {
    c.scene = scene
    c.player = player
    c.combat = combat
    c.expireAt = time.Now().Add(c.ttl)
}
```

### 7.2 并行Tool调用

```go
// executeToolsParallel 并行执行多个Tool
func (l *ReActLoop) executeToolsParallel(ctx context.Context, calls []ToolCall) []ToolResult {
    results := make([]ToolResult, len(calls))

    var wg sync.WaitGroup
    for i, call := range calls {
        wg.Add(1)
        go func(idx int, tc ToolCall) {
            defer wg.Done()
            results[idx] = l.executeTool(ctx, tc)
        }(i, call)
    }
    wg.Wait()

    return results
}
```

### 7.3 流式输出

```go
// StreamOutput 流式输出接口
type StreamOutput interface {
    Write(p []byte) (n int, err error)
    Flush() error
}

// streamResponse 流式输出响应
func (l *ReActLoop) streamResponse(ctx context.Context, content string) {
    // 分块输出，提升用户体验
    chunks := splitContent(content, 50) // 每50字符一块
    for _, chunk := range chunks {
        select {
        case <-ctx.Done():
            return
        default:
            l.output.Write([]byte(chunk))
            l.output.Flush()
            time.Sleep(50 * time.Millisecond) // 模拟打字效果
        }
    }
}
```

## 8. 完整流程示例

### 8.1 战斗场景示例

```
[Phase: OBSERVE]
收集状态:
- 当前场景: 地下城走廊
- 玩家HP: 28/32
- 战斗状态: 第3轮，玩家回合
- 敌人: 哥布林x2 (HP: 5/7, 3/7)

[Phase: THINK]
玩家输入: "我用剑攻击左边的哥布林"

主Agent推理:
1. 识别意图: 战斗攻击
2. 确认目标: 哥布林_A
3. 确认武器: 长剑
4. 需要调用: Combat Agent

[Phase: ACT]
Tool调用:
- execute_attack(attacker: player, target: goblin_a, weapon: longsword)

引擎执行:
- 攻击掷骰: d20+5 = 18
- 目标AC: 13
- 结果: 命中
- 伤害: 1d8+3 = 8
- 哥布林_A HP: 5-8 = -3 (死亡)

[Phase: THINK]
主Agent处理结果:
- 更新状态摘要
- 生成叙事响应

[Phase: ACT]
输出:
"你挥剑斩向哥布林！攻击掷骰18，命中！
你的长剑划过哥布林的身体，造成8点伤害！
哥布林倒地不起，不再动弹。
还有一个哥布林站在你面前，它看起来很害怕。"

[Phase: WAIT]
等待玩家输入...
```

### 8.2 探索场景示例

```
[ReAct Phase: OBSERVE]
引擎Phase: Exploration
收集状态:
- 当前场景: 古老神殿入口
- 玩家HP: 28/32
- 任务: 寻找失落的圣杯
- 附近NPC: 老旅者

[ReAct Phase: THINK]
玩家输入: "我想和那个老旅者交谈"

主Agent推理:
1. 识别意图: 社交互动
2. 确认目标: 老旅者
3. 需要调用: NPC Agent

[ReAct Phase: ACT]
Tool调用:
- interact_with_npc(npc_id: old_traveler, action: talk)

引擎执行:
- 获取NPC态度: 友善
- 触发对话事件
- 返回对话选项

[ReAct Phase: THINK]
主Agent处理:
- 生成NPC对话内容
- 提供玩家选择

[ReAct Phase: ACT]
输出:
"你走向那位老旅者，他抬头看了你一眼，露出了友善的微笑。

'啊，又一个冒险者。你是来寻找传说中的圣杯的吧？'
老旅者抚着胡子说道。

'我可以告诉你一些线索，但你得先帮我一个小忙。'
他指了指神殿入口旁边的雕像。

'那雕像上的宝石被人偷走了，如果你能帮我找回来，
我就告诉你进入神殿的秘密通道。'

你想要如何回应？
1. 答应帮助他找回宝石
2. 询问更多关于宝石的信息
3. 威胁他说出秘密
4. 尝试用其他方式说服他"

[ReAct Phase: WAIT]
等待玩家输入...
```
