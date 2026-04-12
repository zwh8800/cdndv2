# AGENTS.md

This file provides guidance to Qoder (qoder.com) when working with code in this repository.

## Project Overview

**cdndv2** is a D&D (Dungeons & Dragons) LLM-based game engine written in Go 1.24.2. It integrates with OpenAI's language models to create an interactive tabletop RPG experience using a multi-Agent architecture with a ReAct (Reasoning + Acting) loop pattern.

**Key Design Principle**: The `game_engine` NEVER performs game logic calculations. All D&D 5e rule execution (damage calculation, skill checks, leveling, etc.) is delegated to the `dnd-core` engine (`github.com/zwh8800/dnd-core`). The game_engine serves purely as an orchestrator and LLM interface.

## Quick Start

### Build and Run

```bash
# Build
go build -o cdndv2 .

# Run (requires OpenAI API key)
export OPENAI_API_KEY=sk-your-key-here
export OPENAI_MODEL=gpt-4o  # Optional, defaults to gpt-4o
export OPENAI_BASE_URL=...  # Optional, for proxies
./cdndv2
```

### Testing

No test files currently exist in the repository. When adding tests, use Go's standard testing framework with `go test ./...` to run all tests.

### Dependencies

The project uses a local path replacement for dnd-core:
```
replace github.com/zwh8800/dnd-core => ../dnd-core
```

Ensure the `../dnd-core` directory exists and is properly set up.

## Architecture

### High-Level Flow

```
Player Input → ReAct Loop → Main Agent (DM) → SubAgents → Tools → D&D Core Engine → Response
```

### Core Components

1. **ReAct Loop** (`game_engine/react_loop.go`)
   - State machine with phases: Observe → Think → Act → Wait/End
   - Enforces max iterations (default 10) to prevent infinite loops
   - Observe: Collects game state via `CollectSummary()`
   - Think: Calls Main Agent via LLM
   - Act: Executes Tool/SubAgent calls

2. **Main Agent** (`game_engine/agent/main_agent.go`)
   - Central DM (Dungeon Master) decision-maker
   - Builds dynamic system prompts with game state, Tools, SubAgents
   - Calls OpenAI LLM with function calling
   - Parses LLM responses to extract Tool calls, SubAgent calls, narrative content

3. **SubAgents** (`game_engine/agent/*.go`)
   - Character Agent: Role creation/management
   - Combat Agent: Battle flow control
   - Rules Agent: Rule checks/spell execution
   - Narrative/NPC/Memory Agents: Planned for Phase 3+

4. **Tool Registry** (`game_engine/tool/registry.go`)
   - Maps ~92 D&D engine APIs to LLM-callable Tools
   - Maintains indices by Tool name, Agent, and category
   - Converts Tools to OpenAI function-calling format
   - Executes Tools and formats results for LLM consumption

5. **LLM Abstraction** (`game_engine/llm/`)
   - Interface-based: `LLMClient` with `Complete()` and `Stream()`
   - OpenAI implementation in `game_engine/llm/openai/`
   - Configurable model, API key, BaseURL, temperature, maxTokens

6. **State Management** (`game_engine/state/`)
   - `GameSummary`: Wrapper around D&D engine state
   - `CollectSummary()`: Queries engine and formats for LLM context
   - LLM-friendly formatting utilities

### Directory Structure

```
cdndv2/
├── main.go                          # CLI entry point
├── go.mod                           # Go module (v1.24.2)
├── game_engine/
│   ├── engine.go                    # GameEngine facade
│   ├── react_loop.go                # ReAct loop state machine
│   ├── agents.go                    # SubAgent factory & Tool registration
│   ├── agent/                       # Multi-agent system
│   │   ├── agent.go                 # Base interfaces
│   │   ├── main_agent.go            # Main DM Agent
│   │   └── *_agent.go               # SubAgents
│   ├── tool/                        # Tool system
│   │   ├── tool.go                  # Tool interface
│   │   ├── registry.go              # ToolRegistry
│   │   └── *_tools.go               # Tool implementations
│   ├── llm/                         # LLM client abstraction
│   │   ├── client.go                # LLMClient interface
│   │   └── openai/                  # OpenAI implementation
│   ├── state/                       # Game state management
│   └── prompt/                      # Prompt templates
└── docs/design/                     # Comprehensive design docs
    ├── architecture.md              # System architecture (456 lines)
    ├── agent-design.md              # Multi-agent design (884 lines)
    ├── react-loop.md                # ReAct loop design (1059 lines)
    └── tool-design.md               # Tool design (1070 lines)
```

## Development Phases

| Phase | Focus | Status |
|-------|-------|--------|
| Phase 1 | Core framework (Tool Registry, Main Agent, ReAct Loop) | Complete |
| Phase 2 | Character + Combat + Rules agents | In Progress |
| Phase 3 | Narrative, NPC, Memory agents | Planned |
| Phase 4 | Optimization & error handling | Planned |

See `.qoder/specs/` for detailed implementation plans.

## Key Patterns & Conventions

### Adding a New SubAgent

1. Create `game_engine/agent/new_agent.go` implementing `SubAgent` interface
2. Define system prompt template in `game_engine/prompt/`
3. Register Tools for the agent in `game_engine/agents.go`
4. Add to factory function `createSubAgents()`

### Adding New Tools

1. Create Tool struct implementing `Tool` interface
2. Implement `Execute()` to call D&D engine API
3. Register in `registerAgentTools()` in `game_engine/agents.go`
4. Associate with relevant Agents

### Interface-Based Design

All core components use interfaces:
- `Agent`, `SubAgent` in `game_engine/agent/agent.go`
- `Tool` in `game_engine/tool/tool.go`
- `LLMClient` in `game_engine/llm/client.go`

This enables testing and swapping implementations easily.

### Context Passing

`AgentContext` carries state, history, and engine reference throughout the system. Agents access game state through context rather than tight coupling.

## Important Design Documents

Comprehensive design documentation is available in `docs/design/`:
- **architecture.md**: Full system architecture with diagrams
- **agent-design.md**: Multi-Agent patterns, interfaces, and prompts
- **react-loop.md**: State machine and loop phase explanations
- **tool-design.md**: Tool registry and schema definitions

These documents should be consulted before making architectural changes.

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `OPENAI_API_KEY` | OpenAI API key | Required |
| `OPENAI_MODEL` | Model to use | `gpt-4o` |
| `OPENAI_BASE_URL` | Custom API endpoint | OpenAI default |
