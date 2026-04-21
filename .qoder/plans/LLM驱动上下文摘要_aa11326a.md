# LLM 驱动上下文摘要改进方案

## 背景

当前 `summarizeSegment` 使用简单字节截断（用户消息 200 字节、助手消息 300 字节），存在语义丢失、句子中间断裂、信息不可恢复等问题。业界主流 AI Agent 系统（Claude Code、Codex CLI、OpenCode、OpenHands、LangChain Deep Agents）均使用 LLM 生成结构化摘要。

## 设计要点

### 1. 压缩器增加 LLMClient 依赖

`ContextCompressor` 新增 `LLMClient` 字段，用于调用 LLM 生成摘要。当 LLMClient 为 nil 时回退到当前的启发式截断（保持向后兼容）。

### 2. 两级压缩策略（参考 OpenCode）

在调用 LLM 摘要之前，先做轻量级修剪（prune），减少发送给摘要 LLM 的 token 量：

- **Level 1 - Prune**：修剪旧的大型工具输出（>500 token 的工具结果截断到前 100 token + 元信息标注），查询类工具结果直接移除只保留工具名
- **Level 2 - LLM Summarize**：将 pruned 后的旧轮次消息发送给 LLM，用结构化 prompt 生成摘要

### 3. D&D 专属结构化摘要 Prompt

参考 Claude Code 的 checklist 式设计，定制 D&D 游戏领域 prompt：

```
你是一个D&D游戏会话的上下文压缩器。请将以下对话历史压缩为结构化摘要，用于让DM（地下城主）AI继续游戏。

必须包含以下部分（如果存在）：
1. 【情节进展】当前故事线和关键剧情事件
2. 【玩家行动】玩家做出的重要决策和行动
3. 【战斗/遭遇】进行中或已完成的战斗，关键结果（伤害、击杀、状态效果）
4. 【NPC交互】与NPC的重要对话和关系变化
5. 【物品/状态变更】获得/失去的物品、状态变化、位置移动
6. 【待处理事项】进行中但未完成的任务或悬而未决的情况
7. 【关键ID引用】涉及的重要实体ID（角色、场景、物品等）

要求：
- 保留所有实体ID（格式如 01HXXXXXX）
- 保留数值结果（骰子结果、HP变化等）
- 用简洁的条目式表达，不要叙事性文字
- 如果某个部分没有相关内容，跳过该部分
```

### 4. 摘要 LLM 调用配置

- 使用与主 Agent 相同的 LLMClient（无需额外创建）
- Temperature 设为 0（确保摘要确定性）
- MaxTokens 限制为 2000（摘要不应过长）
- 不传递 Tools（摘要不需要工具）

## 实现任务

### Task 1: ContextCompressor 增加 LLMClient 依赖

文件: `game_engine/llm/context_compressor.go`

- 结构体新增 `LLMClient LLMClient` 字段
- `DefaultContextCompressor()` 签名改为接受 `LLMClient` 参数
- 新增摘要 system prompt 常量 `compactionSystemPrompt`
- 新增摘要 user prompt 构建方法 `buildSummarizationPrompt(messages []Message) string`

### Task 2: 实现两级压缩 - Level 1 Prune

文件: `game_engine/llm/context_compressor.go`

- 新增 `pruneOldMessages(messages []Message) []Message` 方法
  - 查询类工具结果替换为 `[查询结果已省略，可通过 GameSummary 恢复]`
  - 大型工具输出（>500 token）截断保留前 100 token + `...[已截断，原始约 N tokens]`
  - 工具调用参数中的大 JSON（>200 token）截断

### Task 3: 实现两级压缩 - Level 2 LLM Summarize

文件: `game_engine/llm/context_compressor.go`

- 新增 `summarizeWithLLM(ctx context.Context, messages []Message) (string, error)` 方法
  - 将消息格式化为可读文本（角色: 内容）
  - 调用 LLMClient.Complete() 生成摘要
  - 错误处理：LLM 调用失败时回退到启发式摘要
- 修改 `compressOldMessages()` 调用链：
  1. 先 prune
  2. 再调用 LLM 摘要
  3. 失败回退到当前的 `summarizeSegment` 逻辑

### Task 4: 异步压缩适配 context.Context

文件: `game_engine/llm/context_compressor.go`

- `CompressHistory` 签名改为 `CompressHistory(ctx context.Context, messages []Message)`（LLM 调用需要 context）
- `StartAsyncCompress` 签名改为接受 `context.Context`
- 内部 goroutine 使用传入的 context（或 `context.Background()`）

### Task 5: 调用链适配

文件: `game_engine/engine.go`
- `DefaultContextCompressor()` 调用传入 `llmClient`

文件: `game_engine/react_loop.go`
- `maybeCompressHistory()` 传递 context 给 `StartAsyncCompress`
- `CompressHistory` 调用处传递 context

### Task 6: 编译验证

运行 `go build ./...` 确保编译通过。
