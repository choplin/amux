# Amux Development Principles

## Architecture Layers

### 1. CLI Commands Layer (`internal/cli/commands/`)

**Purpose**: Thin layer for CLI interaction only.

**Responsibilities**:
- Parse command-line arguments and flags
- Call business logic methods
- Display results using UI helpers
- Handle CLI-specific errors (e.g., wrong number of arguments)

**What NOT to do**:
- Business logic
- Direct file I/O
- Complex data transformations
- Configuration parsing beyond CLI flags
- Decision making based on data types or states

**Example**:
```go
// GOOD: Thin command that delegates to business logic
func runSession(cmd *cobra.Command, args []string) error {
    agentID := args[0]

    sessionManager, err := session.SetupManager()
    if err != nil {
        return err
    }

    sess, err := sessionManager.CreateSession(ctx, session.Options{
        AgentID:     agentID,
        WorkspaceID: workspaceID,
        Command:     runCommand,    // from CLI flag
        Environment: runEnv,        // from CLI flag
    })
    if err != nil {
        return err
    }

    ui.Success("Session created: %s", sess.ID())
    return nil
}

// BAD: Command doing business logic
func runSession(cmd *cobra.Command, args []string) error {
    // DON'T load config files
    cfg, err := config.LoadProjectConfig(projectRoot)

    // DON'T check types and make decisions
    if agent.Type == "tmux" {
        tmuxAgent := cfg.GetTmuxAgent(agentID)
        command = tmuxAgent.GetCommand()
    }

    // DON'T merge data from multiple sources
    env := mergeEnvironment(agentEnv, cliEnv)
}
```

### 2. Business Logic Layer (`internal/core/`)

**Purpose**: Core application logic and domain models.

**Responsibilities**:
- Implement business rules
- Manage state and lifecycle
- Coordinate between different components
- Make decisions based on data

**Principles**:
- Managers handle complex operations and coordination
- Types/models are simple data structures with minimal methods
- Configuration is read-only static data

**Example**:
```go
// session.Manager handles all session creation logic
func (m *Manager) CreateSession(ctx context.Context, opts Options) (Session, error) {
    // Get agent configuration
    agent, err := m.configManager.GetAgent(opts.AgentID)
    if err != nil {
        return nil, fmt.Errorf("agent not found: %w", err)
    }

    // Make decisions based on agent type
    switch agent.Type {
    case config.AgentTypeTmux:
        return m.createTmuxSession(ctx, opts, agent)
    default:
        return nil, fmt.Errorf("unsupported agent type: %s", agent.Type)
    }
}
```

### 3. Configuration Layer (`internal/core/config/`)

**Purpose**: Read and provide access to configuration data.

**Responsibilities**:
- Load configuration from files
- Validate configuration
- Provide type-safe access to configuration
- Cache configuration in memory

**What NOT to do**:
- Execute business logic
- Make decisions (beyond validation)
- Manage lifecycle of other components

**Example**:
```go
// config.Manager provides access to configuration
func (m *Manager) GetAgent(id string) (*Agent, error) {
    cfg, err := m.Load()
    if err != nil {
        return nil, err
    }

    agent, exists := cfg.Agents[id]
    if !exists {
        return nil, fmt.Errorf("agent %q not found", id)
    }

    return &agent, nil
}

// Provide type-specific getters for type safety
func (m *Manager) GetTmuxAgent(id string) (*TmuxAgent, error) {
    agent, err := m.GetAgent(id)
    if err != nil {
        return nil, err
    }

    if agent.Type != AgentTypeTmux {
        return nil, fmt.Errorf("agent %q is not a tmux agent", id)
    }

    return &TmuxAgent{Agent: agent}, nil
}
```

## Key Principles

### 1. Separation of Concerns

- Each layer has a clear, single responsibility
- Dependencies flow downward (CLI → Business Logic → Data)
- No circular dependencies

### 2. Static vs Dynamic

- **Static**: Configuration files, agent definitions
  - Read once, used many times
  - No state management needed
  - Simple getters are sufficient

- **Dynamic**: Sessions, workspaces
  - Have lifecycle (create, start, stop, remove)
  - Need state management
  - Require managers for coordination

### 3. Manager Pattern

Use managers when you need:
- State management
- Lifecycle management
- Coordination between components
- Complex operations

Don't use managers for:
- Simple data access (use direct getters)
- Static configuration (use simple loaders)
- Pure calculations (use functions)

### 4. Error Handling

- Business logic errors should be descriptive
- CLI layer can add context for user-facing messages
- Don't hide errors; wrap them with context

### 5. Testing

- CLI commands: Test argument parsing and output formatting
- Business logic: Test all business rules and edge cases
- Keep test complexity proportional to code complexity

## Anti-patterns to Avoid

1. **Fat Commands**: CLI commands doing business logic
2. **Thin Managers**: Managers that just forward calls without adding value
3. **Over-abstraction**: Creating interfaces/managers for static data
4. **Logic in Models**: Data structures making business decisions
5. **Direct File I/O in Commands**: Commands reading/writing files directly

## Decision Flow

When adding new functionality, ask:

1. Is this parsing CLI input? → Put in command
2. Is this displaying output? → Put in command with UI helpers
3. Is this a business rule? → Put in appropriate manager
4. Is this about data structure? → Put in types/models
5. Is this about configuration? → Put in config package

## Example: Adding a New Agent Type

1. **Config Layer**: Add new agent type constant and validation
2. **Business Logic**: Add handling in session.Manager for the new type
3. **CLI Layer**: No changes needed (still just passes agent ID)

This keeps changes localized and maintains clean separation.
