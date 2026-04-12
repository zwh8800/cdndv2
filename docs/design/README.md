# D&D LLM Game Engine 设计文档

## 文档索引

| 文档 | 说明 |
|------|------|
| [architecture.md](architecture.md) | 整体架构设计，包括组件关系、数据流、目录结构 |
| [agent-design.md](agent-design.md) | Agent系统详细设计，包括各Agent职责和提示词 |
| [tool-design.md](tool-design.md) | Tool定义设计，包括Schema和实现方式 |
| [react-loop.md](react-loop.md) | ReAct循环设计，包括主循环流程和状态管理 |

## 核心设计概要

### 架构模式

采用**多Agent协作**模式：

```
玩家输入 → ReAct循环 → 主Agent(DM) → 子Agent们 → Tool调用 → D&D引擎 → 返回结果 → 生成响应
```

### Agent分层

| 层级 | Agent | 职责 |
|------|-------|------|
| 主控层 | Main Agent (DM) | 意图理解、任务分解、叙事生成、玩家交互 |
| 功能层 | Character Agent | 角色创建、属性管理、升级、休息 |
| | Combat Agent | 战斗流程、回合管理、攻击伤害 |
| | Narrative Agent | 场景管理、物品交互、环境描述 |
| | Rules Agent | 检定、豁免、法术施放 |
| | NPC Agent | NPC行为、态度管理 |
| | Memory Agent | 任务管理、游戏存档 |

### Tool数量统计

| 分类 | API数量 |
|------|---------|
| 游戏会话 | 5 |
| 角色管理 | 10 |
| 战斗系统 | 12 |
| 法术系统 | 10 |
| 检定系统 | 5 |
| 库存管理 | 10 |
| 专长系统 | 5 |
| 场景管理 | 16 |
| 探索系统 | 4 |
| 社交互动 | 2 |
| 任务系统 | 10 |
| 死亡豁免 | 3 |
| **总计** | **92** |

### ReAct循环流程

```
1. OBSERVE: 收集游戏状态、获取玩家输入
2. THINK: 调用主Agent进行分析和决策
3. ACT: 执行Tool调用/子Agent调用，更新状态
4. 循环或等待玩家输入
```

## 目录结构规划

```
game_engine/
├── engine.go              # 主引擎入口
├── react_loop.go          # ReAct循环控制器
├── context.go             # 上下文管理
│
├── agent/
│   ├── agent.go           # Agent接口定义
│   ├── main_agent.go      # 主Agent(DM)
│   ├── character_agent.go # 角色管理Agent
│   ├── combat_agent.go    # 战斗Agent
│   ├── narrative_agent.go # 叙事Agent
│   ├── rules_agent.go     # 规则Agent
│   ├── npc_agent.go       # NPC行为Agent
│   └── memory_agent.go    # 记忆Agent
│
├── tool/
│   ├── registry.go        # Tool注册中心
│   ├── tool.go            # Tool接口定义
│   ├── game_tools.go      # 游戏会话相关Tools
│   ├── actor_tools.go     # 角色管理Tools
│   ├── combat_tools.go    # 战斗系统Tools
│   ├── spell_tools.go     # 法术系统Tools
│   ├── check_tools.go     # 检定系统Tools
│   ├── inventory_tools.go # 库存管理Tools
│   ├── scene_tools.go     # 场景管理Tools
│   ├── quest_tools.go     # 任务系统Tools
│   └── exploration_tools.go # 探索系统Tools
│
├── llm/
│   ├── client.go          # LLM客户端接口
│   ├── message.go         # 消息格式定义
│   └── response.go        # 响应解析
│
├── prompt/
│   ├── templates.go       # 提示词模板
│   ├── main_system.md     # 主Agent系统提示词
│   └── sub_systems/       # 子Agent系统提示词
│
└── state/
    ├── summary.go         # 状态摘要生成
    └── formatter.go       # 状态格式化
```

## 实现阶段规划

### Phase 1: 核心框架 (预计工作量: 中)
- Tool Registry 基础框架
- Main Agent 基础实现
- ReAct Loop 控制器
- LLM客户端接口

### Phase 2: 核心功能 (预计工作量: 高)
- Character Agent + 角色相关Tools
- Combat Agent + 战斗相关Tools
- Rules Agent + 检定相关Tools

### Phase 3: 扩展功能 (预计工作量: 中)
- Narrative Agent + 场景相关Tools
- NPC Agent + NPC相关Tools
- Memory Agent + 任务/存档Tools

### Phase 4: 优化完善 (预计工作量: 低)
- 错误处理优化
- 性能优化
- 提示词优化

## 关键设计决策

### 1. 为什么选择多Agent模式？

- **复杂度管理**: D&D规则复杂，单Agent难以处理所有场景
- **职责分离**: 每个Agent专注于特定领域，降低提示词复杂度
- **可扩展性**: 新增功能只需添加新Agent
- **可测试性**: 每个Agent可独立测试

### 2. 为什么用ReAct循环？

- **推理透明**: 每一步决策都有明确的推理过程
- **错误恢复**: 可以在循环中处理和恢复错误
- **灵活交互**: 支持多轮Tool调用和子Agent协作

### 3. Tool调用策略

- 所有引擎操作通过Tool完成
- Tool由Registry统一管理
- 支持并行Tool调用优化性能

## 下一步行动

1. 确认设计方案后，开始Phase 1的实现
2. 选择LLM提供商（OpenAI/Anthropic/本地模型）
3. 实现基础的Tool框架
4. 实现Main Agent和ReAct循环
5. 完成最小可运行版本（角色创建 + 简单交互）
