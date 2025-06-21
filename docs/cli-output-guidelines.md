# CLI Output Design Guidelines

This document establishes the design principles and patterns for consistent CLI output across all amux commands.

## Design Principles

### 1. Minimalism

- Use visual elements sparingly and purposefully
- Every icon, color, or decoration must serve a clear function
- Prefer clean, undecorated text for most output

### 2. Hierarchy

- Create clear visual distinction between different information levels
- Use indentation to show relationships
- Group related information together

### 3. Consistency

- Apply the same patterns across all commands
- Use the same visual language for similar concepts
- Maintain predictable output formats

### 4. Functionality

- Prioritize readability over decoration
- Ensure output is parseable and scriptable where appropriate
- Support both human and machine consumption

## Visual Elements Usage

### Icons

#### When to Use Icons

- **Section headers**: To identify major output sections
- **Final status**: Success (✅), Error (❌), Warning (⚠️)
- **Entity types**: Workspace (📋), Session (🔄), etc. (sparingly)

#### When NOT to Use Icons

- Consecutive detail lines
- Every line of a list
- Key-value pairs
- Progress steps (unless final status)

#### Icon Guidelines

```text
✅ Success - Use only for final successful completion
❌ Error - Use only for actual errors that need attention
⚠️  Warning - Use only for important warnings
ⓘ  Info - AVOID; rarely needed
📋 Entity icons - Use only in headers, not in lists
```

### Colors

#### Color Palette

- **Default**: Terminal default (no color)
- **Success**: Green (#00D26A) - Sparingly, for final results
- **Error**: Red (#F85149) - Only for actual errors
- **Warning**: Yellow (#F0AD4E) - Only for important warnings
- **Info**: Blue (#0969DA) - Rarely needed
- **Dim**: Gray (#6E7781) - For metadata

#### Color Usage Rules

1. Use color only where it adds meaning
2. Never use color as the only distinguisher
3. Ensure output is readable without color
4. Test with both light and dark themes

### Text Formatting

#### Bold

- Command names in help text
- Important values that need emphasis
- Section headers (sparingly)

#### Dim/Gray

- Metadata (IDs, timestamps, ages)
- Supplementary information
- Hints and suggestions

## Output Patterns

### Success Messages

```text
✅ {Action} completed successfully

{Optional details without icons}
```

Example:

```text
✅ Workspace created successfully

Name:   fix-auth-bug
ID:     3
Branch: fix-auth-bug
Path:   /path/to/workspace
```

### Error Messages

```text
❌ Failed to {action}

Error: {specific error message}

{Optional hint without icon}
```

Example:

```text
❌ Failed to create workspace

Error: workspace name 'test' already exists

Use a different name or remove the existing workspace:
  amux ws remove test
```

### List Displays

```text
{Icon} {Entity} ({count})

{Table with headers in caps}
```

Example:

```text
📋 Workspaces (3)

ID  NAME            BRANCH          AGE     STATUS
1   fix-auth-bug    fix-auth-bug    2h      ✓ ok
2   feat-api        feat-api        1d      ✓ ok
3   refactor-cli    main            3d      ⚠ folder missing
```

### Detail Views

```text
{Entity}: {name}

  {Key}:     {value}
  {Key}:     {value}

  {Section}:
    {Key}:   {value}
    {Key}:   {value}
```

Example:

```text
Workspace: fix-auth-bug

  Name:        fix-auth-bug
  Description: Fix authentication timeout issue
  Branch:      fix-auth-bug
  Status:      ✓ Consistent

  Created:     2024-01-15 10:30:15 (2 hours ago)
  Updated:     2024-01-15 12:45:30 (15 minutes ago)

  Paths:
    Worktree:  /path/to/worktree
    Storage:   /path/to/storage
    Context:   /path/to/context.md
```

### Progress Indication

For operations taking less than 3 seconds: No progress indication needed

For longer operations (future):

```text
{Action}...

  {Step 1}...    done
  {Step 2}...    done
  {Step 3}...    in progress
```

### Prompts

```text
{Question or warning}

{Optional details}

{Prompt} [{default}]:
```

Example:

```text
⚠️  This will permanently delete the workspace 'feat-old'

Do you want to continue? [y/N]:
```

## Implementation Guidelines

### Using the UI Package

#### Preferred Functions

- `ui.Success()` - Only for final success messages
- `ui.Error()` - Only for actual errors
- `ui.Warning()` - Only for important warnings
- `ui.Output()` / `ui.OutputLine()` - For most output
- `ui.PrintKeyValue()` - For consistent key-value pairs

#### Avoid Overuse

- Don't use `ui.Info()` for consecutive lines
- Don't add icons to every line
- Don't color every piece of text

### Examples

#### Bad (Current)

```go
ui.Info("ID: %s", ws.Index)
ui.Info("Branch: %s", ws.Branch)
ui.Info("Path: %s", ws.Path)
```

#### Good (Improved)

```go
ui.Success("Workspace created successfully")
ui.OutputLine("")
ui.PrintKeyValue("ID", ws.Index)
ui.PrintKeyValue("Branch", ws.Branch)
ui.PrintKeyValue("Path", ws.Path)
```

## Testing Guidelines

1. Test output with both light and dark terminal themes
2. Verify output is readable without color support
3. Ensure output is parseable for scripting
4. Check alignment and formatting with various data lengths
5. Validate that important information stands out

## Future Considerations

### Animations

- Reserved for operations taking >3 seconds
- Use sparingly and only when it improves UX
- Consider spinner for indefinite waits
- Consider progress bar for determinate operations

### Interactive Elements

- Keep prompts simple and clear
- Provide sensible defaults
- Show what will happen before confirmation
- Allow escape/cancellation

## Accessibility

1. Never rely solely on color to convey information
2. Use clear, descriptive text
3. Maintain consistent patterns
4. Support screen readers where possible
5. Provide alternative text formats when needed
