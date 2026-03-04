# SubNetree -- Copilot Instructions

Network monitoring and infrastructure management platform for home labs and small
networks. Discovers devices, maps topology, monitors health, and provides a React
dashboard.

## Tech Stack

- **Backend**: Go 1.25, standard library HTTP server (Go 1.22+ enhanced ServeMux)
- **Frontend**: React 19, TypeScript 5, Tailwind CSS v3, TanStack Query v5, Recharts v3
- **Database**: SQLite via `modernc.org/sqlite` (pure-Go, no CGO)
- **API**: REST with Swagger (swaggo/swag v1.16), WebSocket (`coder/websocket`)
- **Build**: Make targets (`build`, `test`, `lint`, `swagger`)
- **Lint**: golangci-lint v2 (Go), ESLint + TypeScript strict (frontend)
- **Test**: Go `testing` package + table-driven tests, Vitest + Testing Library (frontend)
- **Package manager**: pnpm v9 (frontend)

## Project Structure

```text
SubNetree/
├── cmd/
│   ├── subnetree/       - Main server entry point
│   └── scout/           - Network scout agent binary
├── internal/            - Private application packages (30+ modules)
│   └── dashboard/       - Embeds frontend via go:embed all:dist
├── pkg/models/          - Shared types and enums (DeviceType, etc.)
├── api/swagger/         - Generated Swagger specs (never edit manually)
├── web/                 - React frontend (Vite + pnpm)
│   └── src/
│       ├── api/         - Typed API client layer
│       ├── components/  - Reusable UI components
│       ├── hooks/       - Custom React hooks
│       ├── pages/       - Route-level page components
│       └── stores/      - Client state (Zustand)
├── plugins/             - Plugin system
├── .github/             - CI workflows and Copilot config
├── .golangci.yml        - Go linter configuration
└── CLAUDE.md            - Claude Code instructions
```

## Build Instructions

**IMPORTANT**: `internal/dashboard/embed.go` uses `//go:embed all:dist`, so
`internal/dashboard/dist/` must exist before any Go build or test command.
Always build the frontend first.

```bash
# Step 1: Install and build frontend (creates web/dist and internal/dashboard/dist)
cd web && pnpm install --frozen-lockfile && pnpm run build
mkdir -p internal/dashboard/dist && cp -r web/dist/. internal/dashboard/dist/

# OR use the Make target (does both steps automatically):
make build-dashboard

# Step 2: Build Go binaries
go build ./...        # Verify build
make build-server     # Build main server binary → bin/subnetree
make build-scout      # Build scout agent binary → bin/scout
make build            # Build all (dashboard + server + scout)
```

## Testing and Validation

```bash
# Go (requires internal/dashboard/dist to exist first)
make test             # go test ./...
make test-race        # go test -race ./...

# Go linting (golangci-lint v2 config: .golangci.yml)
make lint             # golangci-lint run ./...

# Frontend
cd web && pnpm run type-check   # TypeScript strict check
cd web && pnpm run lint         # ESLint
cd web && pnpm run test         # Vitest unit tests

# Swagger spec (must not drift from source)
make swagger          # Requires: go install github.com/swaggo/swag/cmd/swag@v1.16.4
git diff --exit-code api/swagger/
```

## CI Checks (All Must Pass on Every PR)

1. Frontend: `pnpm install --frozen-lockfile && pnpm run type-check && pnpm run lint && pnpm run test && pnpm run build`
2. Go build: `make build-dashboard && go build ./...` (cross-compiled: linux/amd64, linux/arm64, windows/amd64, macos/amd64)
3. Go tests: `go test -race -timeout=10m ./...`
4. Go lint: `golangci-lint run ./...` (config: `.golangci.yml`)
5. Swagger: `make swagger` then `git diff --exit-code api/swagger/`
6. Vulnerability: `govulncheck ./...`

## Code Style

- Conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`
- Co-author tag: `Co-Authored-By: GitHub Copilot <noreply@github.com>`
- Errors wrapped with context: `fmt.Errorf("operation: %w", err)`
- Table-driven tests with `t.Run` and descriptive names
- All lint checks must pass before committing (golangci-lint v2)

## Coding Guidelines

- Fix errors immediately -- never classify them as pre-existing
- Build, test, and lint must pass before any commit
- Never skip hooks (`--no-verify`) or force-push main
- Validate only at system boundaries (user input, external APIs)
- Remove unused code completely; no backwards-compatibility hacks

## Do NOT

- Add `//nolint` directives without fixing the root cause first
- Use `any` in TypeScript or suppress TypeScript errors with `as unknown`
- Edit `api/swagger/` files directly -- always regenerate with `make swagger`
- Add dependencies without updating the lock file
- Use `panic` in library code; return errors instead
- Store secrets, tokens, or credentials in code or config files
- Mark work as complete when known errors remain

