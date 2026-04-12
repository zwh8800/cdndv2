# Phase 2 Implementation Plan: Character Agent, Combat Agent, Rules Agent

## Context

Phase 1已完成核心框架实现，包括Tool Registry、Main Agent、ReAct Loop、LLM Client、State管理等基础设施。Phase 2需要实现三个核心SubAgent及其对应的Tools，使游戏能够处理角色创建、战斗流程、规则检定等核心玩法。

**用户需求：**
- 全部实现Phase 2的所有Agents和Tools
- 每个SubAgent独立调用LLM（有自己的系统提示词和工具集）
- 基于dnd-core引擎API实现

---

## Scope Overview

| 组件 | 数量 | 说明 |
|------|------|------|
| **Character Agent** | 1个Agent + 10个Tools | 角色创建、查询、更新、经验、休息 |
| **Combat Agent** | 1个Agent + 12个Tools | 战斗初始化、回合管理、攻击、伤害治疗 |
| **Rules Agent** | 1个Agent + 15个Tools | 检定、豁免、法术施放、专注管理 |
| **System Prompts** | 3个.md文件 | 各Agent的系统提示词 |

---

## Implementation Steps

### Step 1: Character Tools 实现

**文件:** `game_engine/tool/character_tools.go`

实现以下10个Tools，每个Tool需要：
1. 定义JSON Schema参数
2. 解析params到engine request结构
3. 调用对应的engine API
4. 格式化返回结果

| Tool Name | Engine API | Description |
|-----------|------------|-------------|
| `create_pc` | `CreatePC` | 创建玩家角色 |
| `create_npc` | `CreateNPC` | 创建NPC |
| `create_enemy` | `CreateEnemy` | 创建敌人 |
| `create_companion` | `CreateCompanion` | 创建同伴 |
| `get_actor` | `GetActor` | 获取角色信息 |
| `get_pc` | `GetPC` | 获取PC详情 |
| `list_actors` | `ListActors` | 列出所有角色 |
| `update_actor` | `UpdateActor` | 更新角色状态 |
| `remove_actor` | `RemoveActor` | 移除角色 |
| `add_experience` | `AddExperience` | 添加经验值 |

**关键Engine API签名示例：**
```go
// CreatePC
func (e *Engine) CreatePC(ctx context.Context, req CreatePCRequest) (*CreatePCResult, error)

type CreatePCRequest struct {
    GameID model.ID
    PC     *PlayerCharacterInput
}

type PlayerCharacterInput struct {
    Name          string
    Race          string
    Subrace       string
    Background    string
    Class         string
    Level         int
    Alignment     string
    AbilityScores AbilityScoresInput
    // ... 更多字段
}
```

---

### Step 2: Combat Tools 实现

**文件:** `game_engine/tool/combat_tools.go`

实现以下12个Tools：

| Tool Name | Engine API | Description |
|-----------|------------|-------------|
| `start_combat` | `StartCombat` | 开始战斗 |
| `start_combat_with_surprise` | `StartCombatWithSurprise` | 带突袭的战斗 |
| `get_current_combat` | `GetCurrentCombat` | 获取战斗状态 |
| `get_current_turn` | `GetCurrentTurn` | 获取当前回合 |
| `next_turn` | `NextTurn` | 推进到下一回合 |
| `execute_action` | `ExecuteAction` | 执行动作 |
| `execute_attack` | `ExecuteAttack` | 执行攻击 |
| `move_actor` | `MoveActor` | 移动角色 |
| `execute_damage` | `ExecuteDamage` | 施加伤害 |
| `execute_healing` | `ExecuteHealing` | 治疗 |
| `perform_death_save` | `PerformDeathSave` | 死亡豁免 |
| `end_combat` | `EndCombat` | 结束战斗 |

**关键Engine API签名示例：**
```go
// ExecuteAttack
func (e *Engine) ExecuteAttack(ctx context.Context, req ExecuteAttackRequest) (*ExecuteAttackResult, error)

type ExecuteAttackRequest struct {
    GameID     model.ID
    AttackerID model.ID
    TargetID   model.ID
    Attack     AttackInput
}

type AttackInput struct {
    WeaponID      *model.ID
    SpellID       *string
    IsUnarmed     bool
    IsOffHand     bool
    Advantage     model.RollModifier
    ExtraDamage   []DamageInput
}
```

---

### Step 3: Rules Tools 实现

**文件:** `game_engine/tool/rules_tools.go`

实现以下15个Tools：

**检定类 (5个):**
| Tool Name | Engine API |
|-----------|------------|
| `perform_ability_check` | `PerformAbilityCheck` |
| `perform_skill_check` | `PerformSkillCheck` |
| `perform_saving_throw` | `PerformSavingThrow` |
| `get_passive_perception` | `GetPassivePerception` |
| `short_rest` | `ShortRest` |

**法术类 (10个):**
| Tool Name | Engine API |
|-----------|------------|
| `cast_spell` | `CastSpell` |
| `get_spell_slots` | `GetSpellSlots` |
| `prepare_spells` | `PrepareSpells` |
| `learn_spell` | `LearnSpell` |
| `concentration_check` | `ConcentrationCheck` |
| `end_concentration` | `EndConcentration` |
| `is_concentrating` | 查询方法 |
| `get_concentration_spell` | 查询方法 |
| `start_long_rest` | `StartLongRest` |
| `end_long_rest` | `EndLongRest` |

**关键Engine API签名示例：**
```go
// PerformSkillCheck
func (e *Engine) PerformSkillCheck(ctx context.Context, req SkillCheckRequest) (*SkillCheckResult, error)

type SkillCheckRequest struct {
    GameID    model.ID
    ActorID   model.ID
    Skill     model.Skill
    DC        int
    Advantage model.RollModifier
    Reason    string
}
```

---

### Step 4: System Prompts 创建

创建三个系统提示词文件：

**文件1:** `game_engine/prompt/character_system.md`
- 角色管理专家身份定义
- 可用Tools列表
- 创建角色流程说明
- 输出格式要求

**文件2:** `game_engine/prompt/combat_system.md`
- 战斗系统专家身份定义
- 战斗流程和回合管理
- 可用Tools列表
- 战斗状态机说明

**文件3:** `game_engine/prompt/rules_system.md`
- 规则仲裁专家身份定义
- DC难度参考表
- 法术施放规则
- 专注管理规则

---

### Step 5: SubAgent 实现

#### 5.1 Character Agent

**文件:** `game_engine/agent/character_agent.go`

```go
type CharacterAgent struct {
    registry     *tool.ToolRegistry
    llm          llm.LLMClient
    systemPrompt string
}

// SubAgent interface methods
func (a *CharacterAgent) Name() string { return subAgentNameCharacter }
func (a *CharacterAgent) CanHandle(intent string) bool
func (a *CharacterAgent) Priority() int { return 10 }
func (a *CharacterAgent) Dependencies() []string { return nil }

// Agent interface methods
func (a *CharacterAgent) SystemPrompt(ctx *AgentContext) string
func (a *CharacterAgent) Tools() []tool.Tool
func (a *CharacterAgent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
```

**CanHandle 匹配关键词:** "create_character", "get_actor", "update_actor", "level_up", "rest", "experience"

#### 5.2 Combat Agent

**文件:** `game_engine/agent/combat_agent.go`

- Priority: 20 (高于Character Agent)
- Dependencies: [subAgentNameCharacter]
- CanHandle关键词: "attack", "combat", "turn", "damage", "heal", "move"

#### 5.3 Rules Agent

**文件:** `game_engine/agent/rules_agent.go`

- Priority: 5 (最低，被其他Agent调用)
- Dependencies: nil
- CanHandle关键词: "check", "save", "spell", "concentration", "skill"

---

### Step 6: Registry 注册

**文件:** `game_engine/registry.go` (新建)

创建集中注册函数：

```go
// RegisterPhase2Tools 注册所有Phase 2工具
func RegisterPhase2Tools(registry *tool.ToolRegistry, engine *engine.Engine) {
    // Character Tools
    registry.Register(tool.NewCreatePCTool(engine), []string{subAgentNameCharacter, mainAgentName}, "character")
    registry.Register(tool.NewCreateNPCTool(engine), []string{subAgentNameCharacter}, "character")
    // ... 其他工具

    // Combat Tools
    registry.Register(tool.NewStartCombatTool(engine), []string{subAgentNameCombat}, "combat")
    // ... 其他工具

    // Rules Tools
    registry.Register(tool.NewPerformAbilityCheckTool(engine), []string{subAgentNameRules}, "check")
    // ... 其他工具
}

// RegisterPhase2Agents 注册所有Phase 2 Agent
func RegisterPhase2Agents(reactLoop *ReActLoop, registry *tool.ToolRegistry, llm llm.LLMClient) {
    subAgents := map[string]agent.SubAgent{
        subAgentNameCharacter: agent.NewCharacterAgent(registry, llm),
        subAgentNameCombat:    agent.NewCombatAgent(registry, llm),
        subAgentNameRules:     agent.NewRulesAgent(registry, llm),
    }
    reactLoop.SetAgents(subAgents)
}
```

---

### Step 7: GameEngine 集成

**修改文件:** `game_engine/engine.go`

在 `NewGameEngine` 函数中添加：

```go
// 注册Phase 2工具
RegisterPhase2Tools(ge.registry, ge.dndEngine)

// 创建并注册SubAgents
subAgents := map[string]agent.SubAgent{
    subAgentNameCharacter: agent.NewCharacterAgent(ge.registry, ge.llmClient),
    subAgentNameCombat:    agent.NewCombatAgent(ge.registry, ge.llmClient),
    subAgentNameRules:     agent.NewRulesAgent(ge.registry, ge.llmClient),
}

// 设置到MainAgent
ge.mainAgent.SetSubAgents(subAgents)

// 设置到ReActLoop
ge.reactLoop.SetAgents(subAgents)
```

---

### Step 8: ReActLoop SubAgent调用支持

**修改文件:** `game_engine/react_loop.go`

添加SubAgent调用方法：

```go
// executeSubAgent 执行子Agent
func (l *ReActLoop) executeSubAgent(ctx context.Context, agentName string, req *agent.AgentRequest) (*agent.AgentResponse, error) {
    subAgent, ok := l.agents[agentName]
    if !ok {
        return nil, fmt.Errorf("unknown sub-agent: %s", agentName)
    }
    return subAgent.Execute(ctx, req)
}

// SetAgents 设置可用的子Agents
func (l *ReActLoop) SetAgents(agents map[string]agent.SubAgent) {
    l.agents = agents
}
```

在 `act` 方法中处理 `call_*_agent` 工具调用：

```go
// 检测是否为子Agent调用
if strings.HasPrefix(toolCall.Name, "call_") && strings.HasSuffix(toolCall.Name, "_agent") {
    agentName := strings.TrimPrefix(strings.TrimSuffix(toolCall.Name, "_agent"), "call_")
    subReq := &agent.AgentRequest{
        UserInput: req.UserInput,
        Context:   req.Context,
    }
    result, err := l.executeSubAgent(ctx, agentName, subReq)
    // 处理结果
}
```

---

## Critical Files Summary

### 新建文件

| 文件路径 | 用途 |
|----------|------|
| `game_engine/tool/character_tools.go` | 角色管理10个Tools |
| `game_engine/tool/combat_tools.go` | 战斗系统12个Tools |
| `game_engine/tool/rules_tools.go` | 检定法术15个Tools |
| `game_engine/agent/character_agent.go` | Character Agent实现 |
| `game_engine/agent/combat_agent.go` | Combat Agent实现 |
| `game_engine/agent/rules_agent.go` | Rules Agent实现 |
| `game_engine/prompt/character_system.md` | Character Agent系统提示词 |
| `game_engine/prompt/combat_system.md` | Combat Agent系统提示词 |
| `game_engine/prompt/rules_system.md` | Rules Agent系统提示词 |
| `game_engine/registry.go` | 工具和Agent集中注册 |

### 修改文件

| 文件路径 | 修改内容 |
|----------|----------|
| `game_engine/engine.go` | 添加Phase 2初始化调用 |
| `game_engine/react_loop.go` | 添加SubAgent调用支持 |
| `game_engine/agent/main_agent.go` | 添加SetSubAgents方法 |

---

## Verification

### 编译验证
```bash
cd /Users/wastecat/code/go/cdndv2
go build ./...
```

### 单元测试（可选）
```bash
go test ./game_engine/tool/...
go test ./game_engine/agent/...
```

### 集成测试场景

1. **角色创建流程:**
   - 启动游戏 → Main Agent解析意图 → 调用Character Agent → create_pc Tool → 返回结果

2. **战斗流程:**
   - 玩家输入"攻击哥布林" → Main Agent → Combat Agent → start_combat + execute_attack → 返回战斗结果

3. **检定流程:**
   - 玩家输入"我想检查陷阱" → Main Agent → Rules Agent → perform_skill_check(perception) → 返回检定结果

### 运行测试
```bash
# 设置API Key
export OPENAI_API_KEY=sk-xxx

# 运行游戏
go run main.go

# 测试输入示例
> 创建一个名叫艾尔文的精灵法师
> 我做一个感知检定来发现隐藏的门
> 攻击那个哥布林
```

---

## Implementation Order

推荐实现顺序：

1. **Tools优先:** 先实现所有Tools（不依赖Agent）
   - character_tools.go
   - combat_tools.go
   - rules_tools.go

2. **System Prompts:** 与Tools并行开发
   - character_system.md
   - combat_system.md
   - rules_system.md

3. **Agents实现:** 依赖Tools完成
   - character_agent.go
   - combat_agent.go
   - rules_agent.go

4. **集成和注册:** 依赖上述完成
   - registry.go
   - engine.go 修改
   - react_loop.go 修改

5. **测试验证:** 最后进行
