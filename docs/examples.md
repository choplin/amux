# Amux Examples

Real-world examples and workflows for getting the most out of Amux.

## CLI Workflows

### Interactive Workspace Selection with fzf

```bash
# Select and enter a workspace
amux ws list | fzf | awk '{print $1}' | xargs amux ws cd

# Create alias for quick access
alias acd='amux ws list | fzf | awk "{print \$1}" | xargs amux ws cd'

# Select and remove old workspaces
amux ws list | fzf -m | awk '{print $1}' | xargs -I {} amux ws remove {} --force
```

### Automated Workspace Management

```bash
# Clean up workspaces older than a week
amux ws list --format json | \
  jq -r '.[] | select(.age > 7) | .id' | \
  xargs -I {} amux ws remove {} --force

# Create workspaces for all open issues
gh issue list --json number,title --jq '.[] | @base64' | while read -r issue; do
  _jq() { echo "${issue}" | base64 -d | jq -r "${1}"; }
  number=$(_jq '.number')
  title=$(_jq '.title' | tr ' ' '-' | tr '[:upper:]' '[:lower:]')
  amux ws create "issue-${number}-${title:0:20}"
done
```

### Git Integration

```bash
# Function to create PR-ready workspace
create_pr_workspace() {
  local name=$1
  local base=${2:-main}

  amux ws create "$name" --base-branch "$base"
  amux ws cd "$name"
}

# List workspaces with their current git status
for ws in $(amux ws list --format json | jq -r '.[].path'); do
  echo "=== $(basename "$ws") ==="
  git -C "$ws" status --short
done
```

## AI Agent Workflows

### Parallel Feature Development

```bash
#!/bin/bash
# develop-features.sh - Develop multiple features in parallel

features=(
  "user-authentication"
  "api-rate-limiting"
  "database-migration"
)

# Create workspaces and start agents
for feature in "${features[@]}"; do
  amux ws create "feat-$feature" --description "Implement $feature"
  amux run claude --workspace "feat-$feature" --name "$feature-dev" &
done

# Monitor progress
watch -n 5 'amux ps'
```

### Code Review Workflow

```bash
# Create review workspace for PR
pr_number=$1
amux ws create "review-pr-$pr_number" \
  --branch "pull/$pr_number/head" \
  --description "Review PR #$pr_number"

# Start AI reviewer
amux run claude --workspace "review-pr-$pr_number" \
  --name "pr-review-$pr_number"
```

### Bug Reproduction Environment

```bash
# Create isolated environment for bug investigation
bug_id=$1
commit_hash=$2

# Create workspace at specific commit
amux ws create "bug-$bug_id" --description "Investigate bug #$bug_id"
cd $(amux ws show "bug-$bug_id" | grep "Path:" | awk '{print $2}')
git checkout "$commit_hash"

# Run tests in isolation
amux run pytest --workspace "bug-$bug_id" --command "pytest -xvs"
```

## MCP Integration Examples

### Claude Code Workflow

```javascript
// In Claude Code, manage your entire development workflow

// 1. Start new feature
const ws = await workspace_create({
  name: "feat-oauth",
  description: "Add OAuth2 authentication"
});

// 2. Work on implementation
await resource_workspace_browse({
  workspace_identifier: ws.id,
  path: "src/auth"
});

// 3. Run tests in workspace
await session_run({
  agent_id: "pytest",
  workspace_identifier: ws.id,
  command: "pytest tests/auth"
});

// 4. Create PR when ready
await prompt("prepare-pr", {
  pr_title: "feat: add OAuth2 authentication support"
});
```

### Multi-Agent Collaboration

```javascript
// Coordinate multiple AI agents on related tasks

// Create workspaces for frontend and backend
const [frontend, backend] = await Promise.all([
  workspace_create({ name: "feat-ui-redesign" }),
  workspace_create({ name: "feat-api-v2" })
]);

// Start specialized agents
await Promise.all([
  session_run({
    agent_id: "claude",
    workspace_identifier: frontend.id,
    name: "frontend-dev"
  }),
  session_run({
    agent_id: "gpt",
    workspace_identifier: backend.id,
    name: "backend-dev"
  })
]);

// Monitor both sessions
const sessions = await resource_session_list();
console.log("Active sessions:", sessions);
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Feature Branch Testing
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  isolated-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Amux
        run: |
          curl -L https://github.com/choplin/amux/releases/latest/download/amux-linux-amd64 -o amux
          chmod +x amux
          sudo mv amux /usr/local/bin/

      - name: Create test workspace
        run: |
          amux init
          amux ws create "test-pr-${{ github.event.pull_request.number }}" \
            --branch "${{ github.head_ref }}"

      - name: Run isolated tests
        run: |
          cd $(amux ws show "test-pr-${{ github.event.pull_request.number }}" | grep Path | awk '{print $2}')
          npm install
          npm test

      - name: Cleanup
        if: always()
        run: amux ws remove "test-pr-${{ github.event.pull_request.number }}" --force
```

### Local Testing Pipeline

```bash
#!/bin/bash
# test-all-features.sh - Test all feature branches

# Get all feature branches
for branch in $(git branch -r | grep 'origin/feature/' | sed 's/origin\///'); do
  ws_name=$(echo "$branch" | tr '/' '-')

  echo "Testing $branch..."

  # Create workspace
  amux ws create "$ws_name" --branch "$branch"

  # Run tests
  ws_path=$(amux ws show "$ws_name" | grep Path | awk '{print $2}')
  (
    cd "$ws_path"
    npm install
    npm test
  ) || echo "FAILED: $branch" >> test-failures.log

  # Cleanup
  amux ws remove "$ws_name" --force
done
```

## Custom Workflows

### Project Template

```bash
# Create reusable project template
create_project_template() {
  local template=$1
  local name=$2

  # Create workspace
  amux ws create "$name" --description "New project from $template"

  # Copy template
  ws_path=$(amux ws show "$name" | grep Path | awk '{print $2}')
  cp -r "templates/$template/." "$ws_path/"

  # Initialize
  (
    cd "$ws_path"
    git add .
    git commit -m "chore: initialize from $template template"
  )

  echo "Project created at: $ws_path"
}

# Usage
create_project_template "react-typescript" "my-new-app"
```

### Experiment Branching

```bash
# Safe experimentation function
experiment() {
  local name=$1
  shift  # Remove first argument
  local command=$@

  # Create experiment workspace
  amux ws create "exp-$name-$(date +%s)" \
    --description "Experiment: $name"

  # Run experiment
  ws_path=$(amux ws show "exp-$name-$(date +%s)" | grep Path | awk '{print $2}')
  (
    cd "$ws_path"
    eval "$command"
  )

  echo "Experiment workspace created. Clean up with:"
  echo "amux ws remove exp-$name-* --force"
}

# Usage
experiment "new-build-tool" "npm install -g turbo && turbo init"
```

### Workspace Sync

```bash
# Sync changes between workspaces
sync_workspaces() {
  local source=$1
  local target=$2
  local files=$3

  src_path=$(amux ws show "$source" | grep Path | awk '{print $2}')
  tgt_path=$(amux ws show "$target" | grep Path | awk '{print $2}')

  for file in $files; do
    cp "$src_path/$file" "$tgt_path/$file"
  done

  echo "Synced files from $source to $target"
}

# Usage
sync_workspaces "feat-ui" "feat-api" "src/types/*.ts"
```

## Best Practices

### Naming Conventions

```bash
# Consistent naming for easy filtering
feat-*     # New features
fix-*      # Bug fixes
exp-*      # Experiments
review-*   # Code reviews
test-*     # Testing
docs-*     # Documentation
```

### Workspace Lifecycle

```bash
# 1. Create with clear purpose
amux ws create feat-search --description "Implement full-text search"

# 2. Work in isolation
amux ws cd feat-search

# 3. Regular cleanup
amux ws prune --days 7

# 4. Archive before removing
tar -czf "archive/feat-search-$(date +%Y%m%d).tar.gz" \
  $(amux ws show feat-search | grep Path | awk '{print $2}')
amux ws remove feat-search --force
```

### Session Management

```bash
# Always name your sessions
amux run claude --name "feature-planning"

# Group related sessions
amux run claude --name "auth-frontend" --workspace feat-auth
amux run gpt --name "auth-backend" --workspace feat-auth
amux run gemini --name "auth-tests" --workspace feat-auth

# Monitor session health
amux ps | grep -E "(stuck|idle)" && echo "Sessions need attention!"
```
