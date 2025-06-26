# Container vs Direct Dependencies

## Current State with Container

```mermaid
graph TD
    subgraph "CLI Commands"
        SessionCmds[session commands]
        WorkspaceCmds[workspace commands]
        ConfigCmds[config commands]
    end

    subgraph "Helpers"
        SessionHelpers[session/helpers.go]
        WorkspaceHelpers[workspace/helpers.go]
    end

    subgraph "App Layer"
        Container[app.Container]
    end

    subgraph "Core Managers"
        ConfigManager[config.Manager]
        WorkspaceManager[workspace.Manager]
        SessionManager[session.Manager]
        AgentManager[agent.Manager]
        IDMapper[idmap.IDMapper]
    end

    %% Current dependencies
    SessionCmds --> SessionHelpers
    WorkspaceCmds --> WorkspaceHelpers
    ConfigCmds --> ConfigManager

    SessionHelpers --> Container
    WorkspaceHelpers --> Container

    Container --> ConfigManager
    Container --> WorkspaceManager
    Container --> SessionManager
    Container --> AgentManager
    Container --> IDMapper

    %% Problem: Everything depends on everything
    style Container fill:#ff9999
```

## Without Container (Direct Dependencies)

```mermaid
graph TD
    subgraph "CLI Commands"
        SessionCmds2[session commands]
        WorkspaceCmds2[workspace commands]
        ConfigCmds2[config commands]
    end

    subgraph "Helpers"
        SessionHelpers2[session/helpers.go]
        WorkspaceHelpers2[workspace/helpers.go]
    end

    subgraph "Core Managers"
        ConfigManager2[config.Manager]
        WorkspaceManager2[workspace.Manager]
        SessionManager2[session.Manager]
        AgentManager2[agent.Manager]
        IDMapper2[idmap.IDMapper]
    end

    %% Direct dependencies - only what's needed
    SessionCmds2 --> SessionHelpers2
    WorkspaceCmds2 --> WorkspaceHelpers2
    ConfigCmds2 --> ConfigManager2

    SessionHelpers2 --> ConfigManager2
    SessionHelpers2 --> WorkspaceManager2
    SessionHelpers2 --> SessionManager2
    SessionHelpers2 --> AgentManager2

    WorkspaceHelpers2 --> ConfigManager2
    WorkspaceHelpers2 --> WorkspaceManager2

    %% Cleaner - each helper only imports what it needs
    style SessionHelpers2 fill:#99ff99
    style WorkspaceHelpers2 fill:#99ff99
```

## The Real Problems to Solve

### 1. IDMapper Instance Sharing

```mermaid
graph LR
    subgraph "Problem"
        WM1[WorkspaceManager] -->|creates| ID1[IDMapper #1]
        SM1[SessionManager] -->|creates| ID2[IDMapper #2]
    end

    subgraph "Solution Needed"
        ID3[Shared IDMapper]
        WM2[WorkspaceManager] -->|uses| ID3
        SM2[SessionManager] -->|uses| ID3
    end
```

### 2. Complex Initialization Order

```text
1. ConfigManager (needs projectRoot)
2. IDMapper (needs amuxDir from ConfigManager)
3. WorkspaceManager (needs ConfigManager + IDMapper)
4. AgentManager (needs ConfigManager)
5. SessionManager (needs all above)
```

## Key Insights

1. **Container makes every command depend on ALL managers** (even unused ones)
2. **IDMapper duplication is the main concrete problem** to solve
3. **Initialization order is complex** but happens once per command
4. **Most commands only need 1-2 managers**, not all of them

## Recommendation

Remove Container and solve the real problems directly:

1. Share IDMapper instance between managers
2. Accept some initialization code duplication for clarity
3. Let each command import only what it needs
