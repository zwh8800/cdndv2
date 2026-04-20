# 集成测试问题分析报告

> 基于 `game_engine/engine_test.log` 的完整分析，记录于 2026-04-21

---

## 一、严重问题（Critical）

### 1. actor_id 传参错误：SubAgent 之间无法共享角色 ID

**现象**：inventory_agent 添加装备时，所有 12 次 `add_item` 调用均返回 `Error: entity not found`。

**根因**：character_agent 创建角色后返回的 actor_id 是 ULID 格式（如 `01KPNT8MQG55XA795BM2CKD7P2`），但 inventory_agent 使用角色名称 `"Aldric"` 作为 `actor_id`。dnd-core 要求使用 ULID 格式的 ID 而非角色名称。

**日志证据**：
- Line 666: character_agent 的 create_pc 成功创建 Aldric，返回 `actor_id: 01KPNTBHART2F97SZNGCQY43JS`
- Line 699: inventory_agent 调用 `add_item` 时使用的 `actor_id: "Aldric"`（名称而非 ID）
- Line 729-751: 全部 12 次 add_item 调用均返回 `Error: entity not found`

**影响**：角色装备完全无法添加，创建后的角色无法配置。

**建议修复**：
1. SubAgent 之间应共享关键实体 ID（如 actor_id）。在 Router 分发任务后，应将前序 Agent 的关键输出（如创建的角色 ID）注入到后续 Agent 的上下文中。
2. 在 SubAgent 的 system prompt 中尽可能提供已有角色的 ID 信息。
3. add_item 等 API 应考虑支持名称查找（fallback），或在 API 文档中更明确要求 ULID。

---

### 2. 重复创建角色：character_agent 二次调用 create_pc

**现象**：玩家首次输入属性后，character_agent 调用 `create_pc` 创建了 Aldric（ID: `01KPNT8MQG55XA795BM2CKD7P2`）。但当玩家输入"完成角色创建"后，character_agent 又调用 `create_pc` 创建了第二个 Aldric（ID: `01KPNTBHART2F97SZNGCQY43JS`），且属性值不同（str:15 vs 原来的 str:16）。

**日志证据**：
- Line 92: 第一次 create_pc，`ability_scores.strength: 16`，返回 `actor_id: 01KPNT8MQG55XA795BM2CKD7P2`
- Line 658: 第二次 create_pc，`ability_scores.strength: 15`，返回 `actor_id: 01KPNTBHART2F97SZNGCQY43JS`

**影响**：游戏中存在两个同名角色，且第二次创建覆盖了玩家原始设定的属性值。

**建议修复**：
1. character_agent 在收到"完成角色创建"类指令时，应先查询现有角色（用 `list_actors` 或 `get_pc`）而非重新创建。
2. `create_pc` 应有幂等性保护：检测到同名角色已存在时，应更新而非重复创建。
3. 在 SubAgent 的 system prompt 中注入已知角色列表信息。

---

### 3. narrative_agent 使用错误的 actor_id

**现象**：narrative_agent 调用 `move_actor_to_scene` 时使用 `"actor_id": "aldric_guard_captain"`，这既不是正确的 ULID 也不是角色名称格式。

**日志证据**：
- Line 906: `move_actor_to_scene` 调用，`actor_id: "aldric_guard_captain"`，返回 `Error: entity not found`

**影响**：角色无法被移动到场景中，场景与角色脱节。

**建议修复**：与问题 1 同属 SubAgent 间信息不共享的根本原因。需要在委派链中传递已有角色的 ID。

---

## 二、高危问题（High）

### 4. 种族名称中英文不一致

**现象**：`create_pc` 接受 `"human"`（小写英文）创建成功，但 `get_race` 用 `"Human"`（首字母大写英文）查询失败，只能用 `"人类"`（中文）查询成功。

**日志证据**：
- Line 92: `create_pc` 用 `"race": "human"` 成功
- Line 154: `get_race("Human")` 返回 `Error: race not found: Human`
- Line 191: `list_races` 显示种族名称为中文 `"人类"`

**影响**：
1. 用户对种族名称的预期不一致，容易导致查询失败。
2. LLM 在调用不同 API 时可能使用不同语言/大小写格式，增加出错概率。
3. create_pc 和 get_race 的种族名称空间不一致。

**建议修复**：
1. 统一 dnd-core 中种族/职业等实体的命名规范（建议中英文都支持，或全部统一为一种）。
2. 在 Tool 的 description 中明确说明预期的参数格式和可接受的值。
3. 考虑在 get_race 等查询类 API 中增加模糊匹配或别名支持。

---

### 5. create_pc 忽略玩家设定的 HP

**现象**：玩家明确要求 HP=20，但 `create_pc` 返回的角色 HP 为 12/12。系统按规则计算了战士1级 HP（10 + CON modifier 2 = 12），忽略了玩家的自定义设置。

**日志证据**：
- Line 92: create_pc 返回 `hit_points: {current: 12, maximum: 12}`，而非玩家要求的 20

**影响**：玩家对角色的控制度受限，DM 规则优先于玩家意愿，但对于"房规"（如高 HP 起始）没有支持途径。

**建议修复**：
1. 在 create_pc 的 Tool schema 中增加 `hit_points_override` 可选参数。
2. 在 system prompt 中告知 DM/LLM 当玩家明确指定 HP 时应传入 override 参数。
3. 或提供单独的 `set_hp` API 用于后续调整。

---

### 6. 角色创建后 speed=0

**现象**：创建的人类战士 speed 为 0，但标准人类种族速度应为 30 尺。

**日志证据**：
- Line 100: create_pc 返回 `"speed": 0`

**影响**：角色的移动速度异常，影响游戏体验。

**建议修复**：
1. create_pc 创建角色后，应自动根据种族数据设置基础速度。
2. 或在 create_pc 的逻辑中，查找种族的 speed 值并填入。

---

## 三、中等问题（Medium）

### 7. SubAgent 之间缺乏状态共享与上下文传递

**现象**：Router 选择了 3 个 Agent 串行执行（character_agent → inventory_agent → narrative_agent），但后续 Agent 无法获取前序 Agent 创建的实体 ID。

**日志证据**：
- Line 643: Router 决定有 3 个 target_agent，顺序执行
- Line 687-689: inventory_agent 的 prompt 中只包含路由意图文本，没有包含 character_agent 的输出结果（如角色 ID）
- inventory_agent 不知道角色的正确 ID
- narrative_agent 同样不知道角色 ID

**影响**：串行 Agent 无法真正协作，第一步的产出无法被后续步骤利用。

**建议修复**：
1. 将前序 SubAgent 的关键输出（如 actor_id、scene_id）注入到后续 SubAgent 的 system prompt 或 user message 中。
2. 在 RouterAgent 的路由结果中增加上下文字段，允许传递关键实体 ID。
3. 或考虑将共享状态存储在 GameSummary 中，让 SubAgent 通过查询获取。

---

### 8. narrative_agent 重复创建场景

**现象**：narrative_agent 创建了两个同名场景"灰岩哨所塔楼"，ID 分别为 `01KPNTD2H383S88XMNFQCW4HG` 和 `01KPNTD9MBC5WCFBRNE8GVFQWR`。

**日志证据**：
- Line 890: 第一次 create_scene 成功，返回 ID `01KPNTD2H383S88XMNFQCW4HG`
- Line 940: 第二次 create_scene 成功，返回 ID `01KPNTD9MBC5WCFBRNE8GVFQWR`

**原因分析**：在 `move_actor_to_scene` 失败后（actor_id 错误），LLM 错误地推断角色尚未创建，然后又调用了一次 `create_scene`，而非重试 move 操作。

**影响**：产生冗余数据，浪费资源。

**建议修复**：
1. SubAgent 在调用失败后应更好地分析错误原因（entity not found 通常意味着 ID 错误而非实体不存在）。
2. 在 SubAgent 的 system prompt 中加入已有实体列表，避免重复创建。
3. create_scene API 应支持幂等检查（如按名称去重）。

---

### 9. 游戏阶段（Phase）始终为 character_creation

**现象**：从第一次输入到最后，游戏状态一直停留在 `character_creation` 阶段。

**日志证据**：
- Line 10: `Phase: character_creation`
- Line 634: 仍然 `Phase: character_creation`

**影响**：游戏阶段不推进，可能影响后续模块的阶段依赖逻辑。

**建议修复**：
1. 在角色创建完成并验证后，自动推进游戏阶段（如从 `character_creation` → `exploration` → `combat`）。
2. 在 GameSummary 中增加阶段推进的条件判断。

---

### 10. MainAgent 历史消息无限增长，Token 消耗快速增加

**现象**：随着交互轮次增加，MainAgent 的消息数从 2 条增长到 20+ 条，导致每次 LLM 调用的 prompt tokens 急剧增加。

**日志证据**：
- Line 32: messageCount: 2 (Round 1)
- Line 127: messageCount: 5 (Round 2)
- Line 556: messageCount: 20 (Round 3, Player Input 3)
- Line 1103: messageCount: 23 (Round 4)

**Token 消耗增长轨迹**：
| 轮次 | promptTokens | 增长 |
|------|-------------|------|
| Router R1 | 1201 | - |
| Router R2 | 1742 | +45% |
| Router R3 | 3464 | +99% |
| Router R4 | 5219 | +51% |
| MainAgent R3 最终 | 7775 | - |

**影响**：
1. API 调用成本快速递增。
2. 可能触及模型的 context window 上限。
3. 新指令可能被历史噪音淹没。

**建议修复**：
1. 实现历史消息摘要（summarization）机制，将多轮工具调用的中间过程压缩。
2. 对已完成委派任务的详细过程（如所有 tool_call/result 对）进行压缩，只保留最终结果。
3. 设置历史消息的最大条数阈值，超限时自动截断或摘要。

---

## 四、低优先级问题（Low）

### 11. LLM 在查询角色状态时重复调用工具

**现象**：Player Input 3（"让我看看我的角色状态"）时，已通过 character_agent 获取了角色详情，但 MainAgent 又重复调用了 `get_pc`、`get_equipment`、`get_inventory`、`get_passive_perception` 等多个工具。

**日志证据**：
- Line 282-347: character_agent 已获取角色详情（包括 ability_scores、HP、AC、background 等）
- Line 373: MainAgent 再次调用 `get_pc`
- Line 417: MainAgent 调用 `get_equipment`
- Line 463: MainAgent 调用 `get_inventory`
- Line 511: MainAgent 调用 `get_passive_perception`

**影响**：浪费 Token 和 API 调用，增加延迟和成本。

**建议修复**：
1. SubAgent 的输出应明确包含所有关键字段，避免 MainAgent 还需要重复查询。
2. MainAgent 应在 system prompt 中被明确告知"已委托 Agent 返回的信息无需再次查询"。

---

### 12. 主循环迭代次数过多

**现象**：单次用户输入"让我看看我的角色状态"导致主循环经历了 12 次迭代才完成响应。

**日志证据**：Line 262: `Phase: 5, Iteration: 12`

**影响**：响应延迟高，用户体验差。

**建议修复**：
1. SubAgent 已返回完整信息时，MainAgent 应直接生成最终回复，避免再次调用多个查询工具。
2. 考虑在 MainAgent 中实现缓存机制，对同一轮对话中已查询的数据不再重复请求。

---

### 13. character_agent 第二次 create_pc 时使用了不同的属性值

**现象**：第一次创建角色时属性为 str:16, dex:12, con:14, int:10, wis:10, cha:8，但 character_agent 第二次创建时使用了 str:15, dex:14, con:13, int:12, wis:10, cha:8（标准数组分配而非玩家原始设定）。

**日志证据**：
- Line 92: 第一次 `ability_scores: {strength:16, dexterity:12, constitution:14, intelligence:10, wisdom:10, charisma:8}`
- Line 658: 第二次 `ability_scores: {strength:15, dexterity:14, constitution:13, intelligence:12, wisdom:10, charisma:8}`

**建议修复**：SubAgent 在执行前应查询并感知已有角色信息，进行增量更新而非重建。

---

### 14. Router 的 target_agents 包含尚不存在的 narrative_agent

**现象**：Router 返回了 3 个 target_agent（character_agent、inventory_agent、narrative_agent），其中 narrative_agent 在设计文档中标注为 Phase 3+ 的功能。

**日志证据**：Line 643: `targetAgentCount: 3`, 包含 narrative_agent

**影响**：启用未完全实现的 Agent 可能导致更多错误。

**建议修复**：在 Router 的可用 Agent 列表中，只包含已实现的 Agent。

---

## 五、问题分类总结

| 类别 | 问题编号 | 描述 |
|------|---------|------|
| **SubAgent 间数据共享** | #1, #7 | Agent 间无法传递关键实体 ID |
| **数据一致性** | #2, #4, #6, #13, #8 | 角色重复创建、命名不一致、属性不一致、场景重复 |
| **参数/API 设计** | #3, #4, #5 | actor_id 格式、种族名称、HP override |
| **性能/成本** | #10, #11, #12 | Token 消耗、重复查询、迭代过多 |
| **游戏逻辑** | #6, #9, #14 | 速度为0、阶段不推进、未实现的 Agent |

---

## 六、优先修复建议

### P0 - 立即修复
1. **SubAgent 间传递实体 ID**（#1, #7）：这是整个系统的根本性设计缺陷。没有实体 ID 共享，多 Agent 协作无法工作。
2. **actor_id 格式统一**（#3）：所有 API 应统一使用 ULID 格式，或在文档中明确标注。

### P1 - 短期修复
3. **防止重复创建角色**（#2, #13）：create_pc 前先查询，或在 Tool description 中提示 LLM "如角色已存在则更新而非重建"。
4. **统一命名空间**（#4）：dnd-core 中种族/职业名称统一为中文或英文，或 API 支持双语查询。
5. **修复 speed=0**（#6）：create_pc 后根据种族数据初始化速度。

### P2 - 中期修复
6. **历史消息压缩**（#10）：实现摘要机制，控制 Token 消耗。
7. **HP override 支持**（#5）：增加自定义 HP 入口。
8. **错误恢复机制**（#8）：SubAgent 遇到 `entity not found` 时应尝试 list_actors 等查找正确 ID。

### P3 - 长期优化
9. **游戏阶段自动推进**（#9）
10. **减少重复查询**（#11, #12）
11. **Router Agent 过滤**（#14）：仅路由到已实现的 SubAgent