<!--
  Scope: AGENTS.md guides the Copilot coding agent and Copilot Chat.
  For code completion and code review patterns, see .github/copilot-instructions.md
  and .github/instructions/*.instructions.md
  For Claude Code, see CLAUDE.md
-->

# SubNetree

Network monitoring and infrastructure management platform for home labs and small
networks. Discovers devices, maps topology, monitors health, and provides a React
dashboard.

## Tech Stack

- **Backend**: Go 1.25, standard library HTTP server (Go 1.22+ enhanced ServeMux)
- **Frontend**: React 19, TypeScript, MUI v5, TanStack Query, Recharts
- **Database**: SQLite (via `modernc.org/sqlite`, pure-Go driver)
- **API**: REST with Swagger (swaggo/swag), WebSocket for real-time events
- **Build**: Make, Docker multi-stage
- **Lint**: golangci-lint v2 (Go), ESLint + TypeScript strict (frontend)
- **Test**: Go `testing` package, Vitest + Testing Library (frontend)

## Build and Test Commands

```bash
# Build
make build          # Build dashboard, server, and scout binaries

# Test
make test           # Run all Go tests
make test-race      # Run Go tests with race detection

# Lint
make lint           # Run golangci-lint

# Swagger
make swagger        # Regenerate API spec from handler annotations

# Frontend
cd web && pnpm install && pnpm run build   # Build frontend
cd web && pnpm run lint                    # Lint frontend
cd web && npx tsc --noEmit                 # TypeScript check
```

## Project Structure

```text
SubNetree/
├── cmd/
│   ├── subnetree/       - Main server entry point
│   └── scout/           - Network scout agent binary
├── internal/
│   ├── auth/            - JWT authentication and authorization
│   ├── config/          - Configuration management (Viper)
│   ├── dashboard/       - Dashboard aggregation module
│   ├── gateway/         - SSH gateway module
│   ├── insight/         - Analytics and anomaly detection
│   ├── llm/             - LLM integration (Ollama)
│   ├── pulse/           - Health monitoring and alerting
│   ├── recon/           - Network reconnaissance and discovery
│   ├── scout/           - Scout agent coordination
│   ├── server/          - HTTP server and middleware
│   ├── settings/        - Runtime settings management
│   ├── store/           - SQLite data access layer
│   ├── vault/           - Secrets management (encrypted at rest)
│   └── ...              - Additional internal packages
├── pkg/models/          - Shared types (DeviceType, enums)
├── api/swagger/         - Generated Swagger specs
├── web/                 - React frontend (Vite + pnpm)
│   └── src/
│       ├── api/         - Typed API client layer
│       ├── components/  - Reusable UI components
│       ├── hooks/       - Custom React hooks
│       ├── pages/       - Route-level page components
│       └── stores/      - Client state (Zustand)
├── plugins/             - Plugin system
├── configs/             - Configuration templates
├── deploy/              - Deployment configurations
├── .github/             - CI workflows and Copilot config
├── Makefile             - Build, test, lint targets
└── CLAUDE.md            - Claude Code instructions
```

## Workflow Rules

### Always Do

- Create a feature branch for every change (`feature/issue-NNN-description`)
- Use conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`
- Run build, test, and lint before opening a PR
- Write table-driven tests with descriptive names
- Wrap errors with context: `fmt.Errorf("operation: %w", err)`
- Fix every error you find, regardless of who introduced it

### Ask First

- Adding new dependencies (check if stdlib covers the need)
- Architectural changes (new packages, major interface changes)
- Database schema migrations
- Changes to CI/CD workflows
- Removing or renaming public APIs

### Never Do

- Commit directly to `main` -- always use feature branches
- Skip tests or lint checks -- even for "small changes"
- Use `--no-verify` or `--force` flags
- Commit secrets, credentials, or API keys
- Add TODO comments without a linked issue number
- Mark work as complete when build, test, or lint failures remain

## Core Principles

These are unconditional -- no optimization or time pressure overrides them:

1. **Quality**: Once found, always fix, never leave. There is no "pre-existing" error.
2. **Verification**: Build, test, and lint must pass before any commit.
3. **Safety**: Never force-push `main`. Never skip hooks. Never commit secrets.
4. **Honesty**: Never mark work as complete when it is not.

## Error Handling

```go
// Wrap errors with context -- every return site should add meaning
if err != nil {
    return fmt.Errorf("load config: %w", err)
}

// Use sentinel errors for caller-distinguishable conditions
var ErrNotFound = errors.New("not found")
if errors.Is(err, ErrNotFound) { ... }
```

## Testing Conventions

```go
// Table-driven tests with descriptive names
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input returns expected output",
            input: "example",
            want:  "result",
        },
        {
            name:    "empty input returns error",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Commit Format

```text
feat: add user authentication endpoint

Implements JWT-based login and token refresh. Tokens expire after 1h.

Closes #42
Co-Authored-By: GitHub Copilot <copilot@github.com>
```

Types: `feat` (new feature), `fix` (bug fix), `refactor` (no behavior change),
`docs` (documentation only), `test` (tests only), `chore` (build/tooling).
