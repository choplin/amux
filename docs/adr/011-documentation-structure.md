# ADR-011: Documentation Structure Strategy

Date: 2025-06-12

## Status

Accepted

## Context

Our documentation has evolved organically, leading to unclear boundaries between different types of
documentation. We currently have:

- Design decisions mixed with descriptive documentation in `architecture.md`
- Rationale and trade-offs documented outside of ADRs
- Unclear guidance on where to document new information
- Redundant information across multiple documents

This creates maintenance challenges:

- Contributors are unsure where to add documentation
- Information becomes outdated in multiple places
- The "why" (decisions) and "what" (current state) are intermingled
- ADRs lose their value as immutable decision records

## Decision

We will establish a clear documentation structure with distinct purposes for each type:

### 1. Architecture Decision Records (ADRs)

**Purpose**: Document the "why" - design decisions, rationale, and trade-offs
**Characteristics**: Immutable once accepted
**Location**: `docs/adr/`
**Content**:

- Context and problem statement
- Considered alternatives
- Decision rationale
- Trade-offs and consequences
- Implementation notes if relevant

### 2. Architecture Documentation

**Purpose**: Document the "what" - current system state and structure
**Characteristics**: Living document that evolves with the system
**Location**: `docs/architecture.md`
**Content**:

- Component overview and responsibilities
- Data flow diagrams
- System architecture diagrams
- Directory structure
- API/Interface documentation
- Extension points
- References to ADRs for rationale

### 3. Component Documentation

**Purpose**: Detailed documentation for specific subsystems
**Characteristics**: Technical reference for developers
**Location**: `docs/{component}.md` (e.g., `docs/mcp.md`)
**Content**:

- Detailed API documentation
- Usage examples
- Configuration options
- Implementation details
- Troubleshooting guides

### 4. Documentation Guide

**Purpose**: Help contributors understand our documentation strategy
**Location**: `docs/README.md`
**Content**:

- Overview of documentation types
- Decision tree for where to document
- Examples of each type
- Contribution guidelines

## Consequences

### Positive

- Clear separation of concerns between decision records and current state
- ADRs remain focused on decisions and rationale
- Architecture documentation can evolve without affecting decision history
- Contributors have clear guidance on where to document
- Reduced redundancy and maintenance burden
- Better discoverability of information

### Negative

- Initial effort required to refactor existing documentation
- Need to educate contributors on the structure
- More documents to maintain (though with clearer boundaries)

### Neutral

- Cross-references needed between documents
- Some duplication acceptable (e.g., brief context in architecture.md with ADR reference)

## Implementation

1. Create `docs/README.md` explaining the documentation structure
2. Extract design decisions from `architecture.md` into new ADRs:
   - ADR-012: Git Worktrees for Workspace Isolation
   - ADR-013: Tmux for Session Management
   - ADR-014: YAML for Configuration
   - ADR-015: File-based Session Store
3. Refactor `architecture.md` to remove rationale/trade-offs
4. Update cross-references between documents
5. Add documentation structure to contributing guidelines

## References

- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
  by Michael Nygard
- [Arc42 Documentation Structure](https://arc42.org/)
- [C4 Model](https://c4model.com/) for architecture diagrams
