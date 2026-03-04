# SubNetree -- Copilot Instructions

Network monitoring and infrastructure management platform for home labs and small
networks. Discovers devices, maps topology, monitors health, and provides a React
dashboard.

## Tech Stack

- **Backend**: Go 1.25, standard library HTTP server (Go 1.22+ enhanced ServeMux)
- **Frontend**: React 19, TypeScript 5, MUI v5 (`@docker/docker-mui-theme` compatible), TanStack Query v5, Recharts v3
- **Database**: SQLite via `modernc.org/sqlite` (pure-Go, no CGO)
- **API**: REST with Swagger (swaggo/swag v1.16), WebSocket (`coder/websocket`)
- **Build**: Make targets (`build`, `test`, `lint`, `swagger`)
- **Lint**: golangci-lint v2 (Go), ESLint + TypeScript strict (frontend)
- **Test**: Go `testing` package + table-driven tests, Vitest + Testing Library (frontend)
- **Package manager**: pnpm (frontend)

## Project Structure

```text
SubNetree/
├── cmd/
│   ├── subnetree/       - Main server entry point
│   └── scout/           - Network scout agent binary
├── internal/            - Private application packages (30+ modules)
├── pkg/models/          - Shared types and enums (DeviceType, etc.)
├── api/swagger/         - Generated Swagger specs
├── web/                 - React frontend (Vite + pnpm)
│   └── src/
│       ├── api/         - Typed API client layer
│       ├── components/  - Reusable UI components (MUI v5)
│       ├── hooks/       - Custom React hooks
│       ├── pages/       - Route-level page components
│       └── stores/      - Client state (Zustand)
├── plugins/             - Plugin system
├── .github/             - CI workflows and Copilot config
└── CLAUDE.md            - Claude Code instructions
```

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

## Available Resources

```bash
make build        # Compile server, scout, and dashboard
make test         # Run all Go tests
make lint         # Run golangci-lint
make swagger      # Regenerate API spec from handler annotations
go build ./...    # Direct build verification
go test ./...     # Direct test run
```

## Do NOT

- Add `//nolint` directives without fixing the root cause first
- Use `any` in TypeScript or suppress TypeScript errors with `as unknown`
- Commit generated files without regenerating them first
- Add dependencies without updating the lock file
- Use `panic` in library code; return errors instead
- Store secrets, tokens, or credentials in code or config files
- Mark work as complete when known errors remain
