# 29. Explicit Branch Operations

Date: 2025-06-21

## Status

Accepted

## Context

The current workspace creation command uses a single `--branch` flag that:
- Takes an existing branch name and uses it for the workspace
- Fails if the branch doesn't exist
- Has no way to specify a custom branch name for new workspaces

This design has several problems:
1. **Ambiguous Intent**: Users can't tell from the command whether a branch will be created or reused
2. **Long Auto-generated Names**: Without specifying `--branch`, workspaces get unwieldy branch names like `amux/workspace-fix-auth-1735000123-a1b2c3d4`
3. **Poor Git Integration**: Terminal prompts become unreadable with long branch names
4. **Confusing Errors**: When a branch doesn't exist, users must manually create it first

## Decision

Replace the single `--branch` flag with two explicit flags:
- `--branch/-b <name>`: Create a new branch with the specified name
- `--checkout/-c <name>`: Use an existing branch

Also simplify `--base-branch` to `--base` for consistency.

The implementation includes:
1. Clear error messages that guide users to the correct flag
2. Comprehensive help text with examples
3. A `BranchExists()` method to check both local and remote branches
4. Explicit `CreateNew` and `UseExisting` fields in `CreateOptions`

## Consequences

### Positive
- **Clear Intent**: Commands explicitly state whether they create or use branches
- **Better UX**: Users can create short, meaningful branch names like `fix-auth`
- **Git Familiarity**: The `-b` flag matches Git's convention for creating branches
- **Helpful Errors**: Error messages tell users exactly which flag to use

### Negative
- **Breaking Change**: Existing scripts using `--branch` will need updates
- **Two Flags**: Users must choose between `-b` and `-c` (but this makes intent explicit)
- **Migration**: Users need to learn the new flags

### Examples

```bash
# Create workspace with new branch
amux ws create fix-auth -b feature/auth-fix

# Create workspace from existing branch
amux ws create fix-auth -c feature/existing-work

# Error messages guide users
$ amux ws create fix -b main
Error: Cannot create branch 'main': already exists. Use -c to checkout existing branch

$ amux ws create fix -c new-feature
Error: Cannot checkout 'new-feature': branch does not exist. Use -b to create new branch
```

This change prioritizes clarity and usability over backward compatibility, which is acceptable since amux is still in its initial development phase.
