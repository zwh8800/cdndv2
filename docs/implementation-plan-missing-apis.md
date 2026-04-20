# 遗漏 API 实现计划

## 概述

本计划旨在将 `dnd-core/pkg/engine` 中尚未在 cdndv2 项目中使用的 81 个 API 方法实现为 Tool，以便 LLM Agent 能够调用这些 D&D 5e 游戏引擎功能。

## 当前状态

- **Engine 总方法数**: ~192 个
- **已实现 Tools**: ~111 个
- **遗漏方法数**: 81 个
- **当前 Tool 覆盖率**: ~58%

## 实现优先级分类

### P0 - 高优先级（核心战斗与角色功能）

这些 API 对游戏流程至关重要，应优先实现：

| 序号 | API 方法 | 功能描述 | 建议 Tool 文件 |
|------|----------|----------|----------------|
| 1 | ExecuteAttack | 执行攻击 | combat_tools.go |
| 2 | ExecuteDamage | 执行伤害 | combat_tools.go |
| 3 | ExecuteHealing | 执行治疗 | combat_tools.go |
| 4 | ExecuteAction | 执行动作 | combat_tools.go |
| 5 | NextTurn | 下一回合 | combat_tools.go |
| 6 | PerformDeathSave | 死亡检定 | combat_tools.go |
| 7 | StabilizeCreature | 稳定生物 | combat_tools.go |
| 8 | AttemptOpportunityAttack | 借机攻击 | combat_tools.go |
| 9 | LevelUp | 升级 | character_tools.go |
| 10 | ApplyExhaustion | 施加力竭 | character_tools.go |
| 11 | RemoveExhaustion | 移除力竭 | character_tools.go |
| 12 | ApplyPoison | 施加中毒 | character_tools.go |
| 13 | RemovePoison | 移除中毒 | character_tools.go |

### P1 - 中优先级（物品与探索功能）

| 序号 | API 方法 | 功能描述 | 建议 Tool 文件 |
|------|----------|----------|----------------|
| 14 | GetWeapon | 获取武器详情 | data_query_tools.go |
| 15 | GetArmor | 获取护甲详情 | data_query_tools.go |
| 16 | GetMount | 获取坐骑详情 | data_query_tools.go |
| 17 | GetMagicItem | 获取魔法物品详情 | data_query_tools.go |
| 18 | ListMounts | 列出可用坐骑 | data_query_tools.go |
| 19 | ListPoisons | 列出毒药 | data_query_tools.go |
| 20 | ListRecipes | 列出配方 | crafting_tools.go |
| 21 | GetRecipe | 获取配方详情 | crafting_tools.go |
| 22 | GetCraftingInfo | 获取制作信息 | crafting_tools.go |
| 23 | Forage | 觅食 | exploration_tools.go |
| 24 | Navigate | 导航 | exploration_tools.go |
| 25 | GetCarryingCapacity | 获取负重能力 | character_tools.go |

### P2 - 较低优先级（数据查询与游戏管理）

| 序号 | API 方法 | 功能描述 | 建议 Tool 文件 |
|------|----------|----------|----------------|
| 26 | ListFeats | 列出专长 | data_query_tools.go |
| 27 | GetFeatDetails | 获取专长详情 | data_query_tools.go |
| 28 | GetBackgroundFeatures | 获取背景特性 | data_query_tools.go |
| 29 | ListGears | 列出工具 | data_query_tools.go |
| 30 | ListTools | 列出工具 | data_query_tools.go |
| 31 | ListTraps | 列出陷阱 | data_query_tools.go |
| 32 | GetTrap | 获取陷阱详情 | exploration_tools.go |
| 33 | GetTool | 获取工具详情 | data_query_tools.go |
| 34 | ListLifestylesData | 列出生活方式 | data_query_tools.go |
| 35 | GetLifestyleInfo | 获取生活方式信息 | character_tools.go |
| 36 | SetLifestyle | 设置生活方式 | character_tools.go |

### P3 - 低优先级（高级功能与环境）

| 序号 | API 方法 | 功能描述 | 建议 Tool 文件 |
|------|----------|----------|----------------|
| 37 | GetGame | 获取游戏详情 | engine_tools.go (new) |
| 38 | SaveGame | 保存游戏 | engine_tools.go (new) |
| 39 | LoadGame | 加载游戏 | engine_tools.go (new) |
| 40 | DeleteGame | 删除游戏 | engine_tools.go (new) |
| 41 | ListGames | 列出游戏 | engine_tools.go (new) |
| 42 | SetPhase | 设置游戏阶段 | engine_tools.go (new) |
| 43 | GetPhase | 获取游戏阶段 | engine_tools.go (new) |
| 44 | GetStateSummary | 获取状态摘要 | engine_tools.go (new) |
| 45 | GetAllowedOperations | 获取允许的操作 | engine_tools.go (new) |
| 46 | Close | 关闭引擎 | engine_tools.go (new) |

### P4 - 特殊功能（法术相关）

| 序号 | API 方法 | 功能描述 | 建议 Tool 文件 |
|------|----------|----------|----------------|
| 47 | CastSpellRitual | 仪式施法 | rules_tools.go |
| 48 | GetConcentrationSpell | 获取专注法术 | rules_tools.go |
| 49 | GetMulticlassSpellSlots | 获取兼职法术位 | rules_tools.go |
| 50 | GetPactMagicSlots | 获取契约魔法术位 | rules_tools.go |
| 51 | RestorePactMagicSlots | 恢复契约魔法术位 | rules_tools.go |
| 52 | IsConcentrating | 检查是否在专注 | rules_tools.go |

### P5 - 骰子与辅助功能

| 序号 | API 方法 | 功能描述 | 建议 Tool 文件 |
|------|----------|----------|----------------|
| 53 | Roll | 一般投骰 | rules_tools.go |
| 54 | RollAbility | 属性投骰 | rules_tools.go |
| 55 | RollAdvantage | 优势投骰 | rules_tools.go |
| 56 | RollDisadvantage | 劣势投骰 | rules_tools.go |
| 57 | RollHitDice | 生命骰投骰 | rules_tools.go |
| 58 | GetSkillAbility | 获取技能关联属性 | rules_tools.go |

### P6 - NPC 与任务扩展

| 序号 | API 方法 | 功能描述 | 建议 Tool 文件 |
|------|----------|----------|----------------|
| 59 | InteractWithNPC | 与NPC互动 | social_tools.go |
| 60 | GetQuestGiverQuests | 获取任务发布者任务 | quest_tools.go |
| 61 | CurseActor | 诅咒角色 | character_tools.go |
| 62 | RemoveCurse | 移除诅咒 | character_tools.go |
| 63 | GetCurses | 获取诅咒列表 | data_query_tools.go |
| 64 | DeleteQuest | 删除任务 | quest_tools.go |
| 65 | SelectFeat | 选择专长 | character_tools.go |
| 66 | RemoveFeat | 移除专长 | character_tools.go |
| 67 | GetActorFeats | 获取角色专长 | character_tools.go |
| 68 | GetActorSheet | 获取角色卡 | character_tools.go |
| 69 | GetDeathSaveStatus | 获取死亡状态 | combat_tools.go |

### P7 - 环境与伤害

| 序号 | API 方法 | 功能描述 | 建议 Tool 文件 |
|------|----------|----------|----------------|
| 70 | ResolveEnvironmentalDamage | 环境伤害处理 | exploration_tools.go |
| 71 | ResolvePoisonEffect | 中毒效果处理 | character_tools.go |
| 72 | GetExhaustionEffects | 获取力竭效果 | character_tools.go |
| 73 | GetExhaustionStatus | 获取力竭状态 | character_tools.go |
| 74 | GetCombatSummary | 获取战斗摘要 | combat_tools.go |
| 75 | LoadMonster | 加载怪物 | monster_tools.go (new) |
| 76 | ApplyBackground | 应用背景 | character_tools.go |
| 77 | ShortRest | 短休 | rules_tools.go |
| 78 | Dismount | 下马 | mount_tools.go |

## 实现步骤

### 阶段 1: 核心战斗系统 (P0)

1. 在 `combat_tools.go` 中添加工具：
   - ExecuteAttackTool
   - ExecuteDamageTool
   - ExecuteHealingTool
   - ExecuteActionTool
   - NextTurnTool
   - PerformDeathSaveTool
   - StabilizeCreatureTool
   - AttemptOpportunityAttackTool
   - GetCombatSummaryTool
   - GetDeathSaveStatusTool

2. 在 `character_tools.go` 中添加工具：
   - LevelUpTool
   - ApplyExhaustionTool
   - RemoveExhaustionTool
   - ApplyPoisonTool
   - RemovePoisonTool

### 阶段 2: 物品与探索 (P1)

3. 在 `data_query_tools.go` 中添加工具：
   - GetWeaponTool
   - GetArmorTool
   - GetMountTool
   - GetMagicItemTool
   - ListMountsTool
   - ListPoisonsTool
   - ListRecipesTool
   - GetRecipeTool
   - GetCraftingInfoTool

4. 在 `exploration_tools.go` 中添加工具：
   - ForageTool
   - NavigateTool
   - GetTrapTool
   - ListTrapsTool

### 阶段 3: 数据查询扩展 (P2)

5. 扩展 `data_query_tools.go`：
   - ListFeatsTool
   - GetFeatDetailsTool
   - GetBackgroundFeaturesTool
   - ListGearsTool
   - ListToolsTool
   - ListLifestylesDataTool
   - GetLifestyleInfoTool

6. 扩展 `crafting_tools.go`：
   - GetRecipeTool

7. 扩展 `character_tools.go`：
   - SetLifestyleTool
   - GetCarryingCapacityTool

### 阶段 4: 游戏管理 (P3)

8. 创建新文件 `engine_tools.go`：
   - GetGameTool
   - SaveGameTool
   - LoadGameTool
   - DeleteGameTool
   - ListGamesTool
   - SetPhaseTool
   - GetPhaseTool
   - GetStateSummaryTool
   - GetAllowedOperationsTool
   - CloseTool

### 阶段 5: 法术系统 (P4)

9. 扩展 `rules_tools.go`：
   - CastSpellRitualTool
   - GetConcentrationSpellTool
   - GetMulticlassSpellSlotsTool
   - GetPactMagicSlotsTool
   - RestorePactMagicSlotsTool
   - IsConcentratingTool

### 阶段 6: 骰子系统 (P5)

10. 扩展 `rules_tools.go`：
    - RollTool
    - RollAbilityTool
    - RollAdvantageTool
    - RollDisadvantageTool
    - RollHitDiceTool
    - GetSkillAbilityTool

### 阶段 7: NPC 与任务 (P6)

11. 扩展 `social_tools.go`：
    - InteractWithNPCTool (完善)

12. 扩展 `quest_tools.go`：
    - GetQuestGiverQuestsTool
    - DeleteQuestTool

13. 扩展 `character_tools.go`：
    - CurseActorTool
    - RemoveCurseTool
    - GetCursesTool
    - SelectFeatTool
    - RemoveFeatTool
    - GetActorFeatsTool
    - GetActorSheetTool
    - ApplyBackgroundTool

### 阶段 8: 环境与特殊功能 (P7)

14. 扩展 `exploration_tools.go`：
    - ResolveEnvironmentalDamageTool

15. 扩展 `character_tools.go`：
    - ResolvePoisonEffectTool
    - GetExhaustionEffectsTool
    - GetExhaustionStatusTool

16. 创建新文件 `monster_tools.go`：
    - LoadMonsterTool

17. 扩展 `rules_tools.go`：
    - ShortRestTool

18. 扩展 `mount_tools.go`：
    - DismountTool

## 新增 Tool 文件建议

### engine_tools.go (新建)

```go
// 建议实现的工具
type GetGameTool struct{...}
type SaveGameTool struct{...}
type LoadGameTool struct{...}
type DeleteGameTool struct{...}
type ListGamesTool struct{...}
type SetPhaseTool struct{...}
type GetPhaseTool struct{...}
type GetStateSummaryTool struct{...}
type GetAllowedOperationsTool struct{...}
type CloseTool struct{...}
```

### monster_tools.go (新建)

```go
// 建议实现的工具
type LoadMonsterTool struct{...}
type GetMonsterStatBlockTool struct{...}
```

## 实现模板

每个 Tool 应遵循以下结构：

```go
type NewToolNameTool struct {
    baseTool
}

func NewToolNameTool(e *engine.Engine) *NewToolNameTool {
    return &NewToolNameTool{
        baseTool: baseTool{
            engine: e,
            name:    "tool_name",
            desc:    "Tool description",
        },
    }
}

func (t *NewToolNameTool) Execute(ctx context.Context, input map[string]any) (string, error) {
    e := t.Engine().(*engine.Engine)
    
    req := engine.NewToolNameRequest{
        // 填充请求字段
    }
    
    result, err := e.NewToolName(ctx, req)
    if err != nil {
        return formatError(err)
    }
    
    return formatResult(result), nil
}

func (t *NewToolNameTool) GetSchema() tool.Schema {
    return tool.Schema{
        Name:        t.Name(),
        Description: t.Description(),
        Parameters:  json.RawMessage(`{...}`),
    }
}
```

## 注册新工具

在 `game_engine/agents.go` 中注册新工具：

```go
func registerAgentTools(reg *tool.Registry, e *engine.Engine) {
    // P0 - 战斗
    reg.RegisterTool(combat_tools.NewExecuteAttackTool(e), combatAgent)
    reg.RegisterTool(combat_tools.NewExecuteDamageTool(e), combatAgent)
    // ... 其他战斗工具
    
    // P1 - 物品与探索
    reg.RegisterTool(data_query_tools.NewGetWeaponTool(e), rulesAgent)
    // ... 其他数据查询工具
    
    // 依此类推
}
```

## 验收标准

1. 每个新 Tool 都应正确调用对应的 Engine 方法
2. 每个 Tool 都应有完整的 JSON Schema 定义
3. 所有 Tool 都应在 Agent Registry 中正确注册
4. 错误处理应一致且用户友好
