# AGENTS.md

This file provides guidance to Qoder (qoder.com) when working with code in this repository.

## Project Overview

**cdndv2** is a D&D 5e LLM game engine in Go. It uses OpenAI function calling with a multi-Agent ReAct loop to run tabletop RPG sessions.

**Fundamental rule**: `game_engine` NEVER performs D&D rule calculations. All mechanics (damage, checks, leveling, etc.) are delegated to the `dnd-core` engine (`github.com/zwh8800/dnd-core`). The game_engine is purely an orchestrator and LLM interface.

## Build, Test & Run

```bash
# Build
go build -o cdndv2 .

# Run (interactive CLI)
export OPENAI_API_KEY=sk-...
./cdndv2

# Run all tests (integration tests require real OpenAI API key)
OPENAI_API_KEY=sk-... go test ./...

# Run a single test
OPENAI_API_KEY=sk-... go test -run TestGameEngineFullFlow -v ./game_engine/

# Run tests with custom model/endpoint
OPENAI_API_KEY=sk-... OPENAI_MODEL=gpt-4o OPENAI_BASE_URL=https://... go test -v -timeout 10m ./game_engine/
```

**Important**: Tests call real LLM APIs and are slow. Use `-timeout 10m` or longer for integration tests. Tests skip automatically when `OPENAI_API_KEY` is not set.

### Dependencies

Local path replacement for dnd-core — the `../dnd-core` directory must exist:
```
replace github.com/zwh8800/dnd-core => ../dnd-core
```

## Architecture

### Request Flow

```
Player Input → GameEngine.ProcessInput()
  → ReActLoop.Run() (state machine)
    → PhaseObserve: CollectSummary() + async compression check
    → PhaseRoute: RouterAgent decides which SubAgents to invoke
    → PhaseThink: MainAgent reasons via LLM (read-only tools + delegate_task)
    → PhaseAct: Execute tool calls, run SubAgent delegations
    → PhaseSynthesize: Combine SubAgent results into narrative
    → PhaseWait/PhaseEnd: Return response to player
```

### Read/Write Tool Separation (Critical Pattern)

MainAgent can ONLY use read-only tools + `delegate_task`. All write operations (creating characters, starting combat, casting spells, etc.) go through SubAgents. This is enforced at the registry level:

- **Read-only tools** are registered with `MainAgentName` in their agent list and have `ReadOnly() == true`
- **Write tools** are registered ONLY with their domain SubAgent(s)
- `delegate_task` is the only way MainAgent triggers state changes

In `game_engine/agents.go`, registration follows this pattern:
```go
// Write — SubAgent only
registry.Register(tool.NewCreatePCTool(engine), []string{agent.SubAgentNameCharacter}, "character")
// Read — MainAgent + SubAgent
registry.Register(tool.NewGetActorTool(engine), []string{agent.SubAgentNameCharacter, agent.MainAgentName}, "character")
```

### SubAgent Base Class Pattern

All 11 SubAgents extend `BaseSubAgent` (in `game_engine/agent/base_sub_agent.go`) which provides:
- System prompt loading from embedded markdown templates
- Template data injection (GameID, PlayerID, GameState, AvailableTools, KnownEntityIDs)
- LLM calling and response parsing
- Intent matching via keywords in `CanHandle()`

Each concrete agent only needs to supply a `SubAgentConfig` struct:
```go
agent.NewBaseSubAgent(SubAgentConfig{
    Name:         SubAgentNameCombat,
    TemplateFile: "combat_system.md",
    Priority:     90,
    Keywords:     []string{"攻击", "战斗", ...},
    ExtraTemplateData: func(ctx *AgentContext) map[string]any { ... },
}, registry, llmClient)
```

### Embedded Prompt Template System

All agent system prompts live in `game_engine/prompt/*.md` and are embedded at build time via `//go:embed *.md`. Templates use Go `text/template` syntax with these standard variables:
- `{{.GameID}}`, `{{.PlayerID}}` — session identifiers
- `{{.GameState}}` — formatted game state summary
- `{{.AvailableTools}}` — list of tool names/descriptions
- `{{.KnownEntityIDs}}` — entity ID mappings for SubAgent coordination

Load and render via `prompt.LoadAndRender("combat_system.md", data)`.

### Async Context Compression

`game_engine/llm/context_compressor.go` prevents context window overflow:
- Estimates token usage with Chinese/English-aware heuristics + dynamic calibration
- Triggers background LLM-driven summarization when usage exceeds 75% of window
- Preserves the 3 most recent conversation rounds intact
- Non-blocking: results are applied on the next `ProcessInput()` call
- Falls back to heuristic truncation when LLM client is nil

### Router Agent

`game_engine/agent/router_agent.go` analyzes player input and produces a `RouterDecision`:
- Determines which SubAgents should handle the request
- Specifies sequential vs parallel execution mode
- Can provide a direct response if no agent delegation is needed

### Entity ID Sharing Between Agents

`AgentContext.KnownEntityIDs` is a `map[string]string` that propagates entity references (actor_id, scene_id) across SubAgent boundaries. The ReActLoop populates it from game state during PhaseObserve, and it's injected into every SubAgent's system prompt so they use correct IDs when calling tools.

## Key Interfaces

| Interface | Location | Methods |
|-----------|----------|---------|
| `Agent` | `game_engine/agent/agent.go` | `Name()`, `Description()`, `SystemPrompt(ctx)`, `Tools()`, `Execute(ctx, req)` |
| `SubAgent` | `game_engine/agent/agent.go` | extends Agent + `CanHandle(intent)`, `Priority()`, `Dependencies()` |
| `Tool` | `game_engine/tool/tool.go` | `Name()`, `Description()`, `ParametersSchema()`, `Execute(ctx, params)`, `ReadOnly()` |
| `LLMClient` | `game_engine/llm/client.go` | `Complete(ctx, req)`, `Stream(ctx, req)` |

## Adding a New SubAgent

1. Create `game_engine/agent/new_agent.go` — construct a `BaseSubAgent` with `SubAgentConfig`
2. Add agent name constant to `game_engine/agent/const.go`
3. Create system prompt template `game_engine/prompt/new_system.md`
4. Register the agent's tools in `registerAgentTools()` in `game_engine/agents.go`
5. Add to `createSubAgents()` map in `game_engine/agents.go`

## Adding a New Tool

1. Create tool struct embedding `EngineTool` (for engine-backed tools) or `BaseTool`
2. Set `readOnly: true` if the tool does not modify game state
3. Implement `Execute(ctx, params)` — call dnd-core engine API, return `*ToolResult`
4. Register in `registerAgentTools()` with appropriate agent associations and category
5. Read-only tools can include `agent.MainAgentName` in their agent list; write tools must not

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `OPENAI_API_KEY` | OpenAI API key | Required |
| `OPENAI_MODEL` | Model name | `gpt-4o` |
| `OPENAI_BASE_URL` | Custom API endpoint | OpenAI default |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` |

## Design Documents

Consult `docs/design/` before making architectural changes:
- `architecture.md` — system architecture
- `agent-design.md` — multi-agent patterns and interfaces
- `react-loop.md` — ReAct state machine
- `tool-design.md` — tool registry and schema definitions
