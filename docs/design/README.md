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
| 功能层 | Character Agent | 角色创建、属性管理、升级、休息、背景、多职业、生活方式 |
| | Combat Agent | 战斗流程、回合管理、攻击伤害、死亡豁免、环境效果 |
| | Narrative Agent | 场景管理、物品交互、环境描述、探索、陷阱 |
| | Rules Agent | 检定、豁免、法术施放、骰子、状态效果（力竭/诅咒/毒药） |
| | NPC Agent | NPC行为、态度管理、怪物加载 |
| | Memory Agent | 任务管理、游戏存档、状态查询、阶段管理 |
| | Movement Agent [新增] | 移动、跳跃、跌落、窒息、遭遇检定 |
| | Mount Agent [新增] | 骑乘系统管理 |
| | Crafting Agent [新增] | 物品制作系统 |
| | Inventory Agent [新增] | 详细库存、装备、魔法物品管理 |
| | Data Query Agent [新增] | 游戏数据查询（种族、职业、法术、装备等） |

### Tool数量统计

| 分类 | API数量 |
|------|---------|
| 游戏会话 | 6 |
| 角色管理 | 10 |
| 角色升级 | 2 |
| 休息系统 | 3 |
| 战斗系统 | 12 |
| 法术系统 | 11 |
| 检定系统 | 5 |
| 库存管理 | 9 |
| 专长系统 | 5 |
| 场景管理 | 14 |
| 探索系统 | 4 |
| 社交互动 | 2 |
| 任务系统 | 10 |
| 死亡豁免 | 3 |
| 背景系统 | 2 |
| 制作系统 | 4 |
| 诅咒系统 | 3 |
| 环境系统 | 2 |
| 力竭系统 | 3 |
| 骑乘系统 | 3 |
| 移动系统 | 5 |
| 毒药系统 | 3 |
| 陷阱系统 | 4 |
| 魔法物品 | 4 |
| 多职业系统 | 2 |
| 生活方式 | 2 |
| 骰子系统 | 5 |
| 数据查询 | 36 |
| 怪物系统 | 1 |
| 信息聚合 | 4 |
| 状态查询 | 3 |
| 阶段管理 | 3 |
| **总计** | **178** |

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
│   ├── agent.go            # Agent接口定义
│   ├── main_agent.go       # 主Agent(DM)
│   ├── character_agent.go  # 角色管理Agent
│   ├── combat_agent.go     # 战斗Agent
│   ├── narrative_agent.go  # 叙事Agent
│   ├── rules_agent.go      # 规则Agent
│   ├── npc_agent.go        # NPC行为Agent
│   ├── memory_agent.go     # 记忆Agent
│   ├── movement_agent.go   # 移动Agent [新增]
│   ├── mount_agent.go      # 骑乘Agent [新增]
│   ├── crafting_agent.go   # 制作Agent [新增]
│   ├── inventory_agent.go  # 库存Agent [新增]
│   └── data_query_agent.go # 数据查询Agent [新增]
│
├── tool/
│   ├── registry.go         # Tool注册中心
│   ├── tool.go             # Tool接口定义
│   ├── game_tools.go       # 游戏会话相关Tools
│   ├── actor_tools.go      # 角色管理Tools
│   ├── combat_tools.go     # 战斗系统Tools
│   ├── spell_tools.go      # 法术系统Tools
│   ├── check_tools.go      # 检定系统Tools
│   ├── inventory_tools.go  # 库存管理Tools
│   ├── scene_tools.go      # 场景管理Tools
│   ├── quest_tools.go      # 任务系统Tools
│   ├── exploration_tools.go # 探索系统Tools
│   ├── background_tools.go # 背景系统Tools [新增]
│   ├── crafting_tools.go   # 制作系统Tools [新增]
│   ├── curse_tools.go      # 诅咒系统Tools [新增]
│   ├── environment_tools.go # 环境系统Tools [新增]
│   ├── exhaustion_tools.go # 力竭系统Tools [新增]
│   ├── mount_tools.go      # 骑乘系统Tools [新增]
│   ├── movement_tools.go   # 移动系统Tools [新增]
│   ├── poison_tools.go     # 毒药系统Tools [新增]
│   ├── trap_tools.go       # 陷阱系统Tools [新增]
│   ├── magic_item_tools.go # 魔法物品Tools [新增]
│   ├── multiclass_tools.go # 多职业Tools [新增]
│   ├── lifestyle_tools.go  # 生活方式Tools [新增]
│   ├── dice_tools.go       # 骰子系统Tools [新增]
│   ├── data_query_tools.go # 数据查询Tools [新增]
│   └── phase_tools.go      # 阶段管理Tools [新增]
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

### Phase 1: 核心框架
- Tool Registry 基础框架
- Main Agent 基础实现
- ReAct Loop 控制器
- LLM客户端接口

### Phase 2: 核心功能
- Character Agent + 角色相关Tools (含背景、多职业、生活方式)
- Combat Agent + 战斗相关Tools (含死亡豁免、环境效果)
- Rules Agent + 检定相关Tools (含法术、骰子、状态效果)
- Inventory Agent + 库存相关Tools (含魔法物品)

### Phase 3: 扩展功能
- Narrative Agent + 场景相关Tools (含探索、陷阱)
- NPC Agent + NPC相关Tools
- Memory Agent + 任务/存档Tools (含阶段管理、状态查询)
- Movement Agent + 移动相关Tools (含环境伤害、窒息)
- Mount Agent + 骑乘相关Tools
- Crafting Agent + 制作相关Tools
- Data Query Agent + 数据查询Tools

### Phase 4: 优化完善
- 错误处理优化
- 性能优化
- 提示词优化
- 新增系统整合 (诅咒、力竭、毒药等)

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
