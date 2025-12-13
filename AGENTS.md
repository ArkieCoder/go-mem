# Agent Instructions for go-mem

## Build & Test
- **Build**: `go build .`
- **Test All**: `go test ./...`
- **Run Single Test**: `go test -v ./internal/<package> -run <TestName>`
- **Lint**: `go vet ./...`

## Code Style & Conventions
- **Formatting**: Standard Go formatting. Run `go fmt ./...` before committing.
- **Imports**: Group standard library first, third-party second, local `go-mem/...` last.
- **Error Handling**: Use `fmt.Errorf("context: %w", err)` to wrap errors.
- **Naming**: `PascalCase` for exported symbols, `camelCase` for internal.
- **Architecture**:
  - `main.go`: Bubble Tea UI model/view/update loop.
  - `internal/game`: Session management and card loading.
  - `internal/state`: Finite State Machine (FSM) and game logic.
  - `internal/scoring`: Scoring rules and JSON persistence.
- **Conventions**: Prefer `lipgloss` for styling. Ensure TUI views handle newlines correctly.

## Agent Permissions & Constraints
- **Git Operations**: Do not run any git operations without getting human approval.
- **System Tools**: Do not run any system tools without getting human approval.
- **Self-Modification**: NEVER update AGENTS.md without human approval.

