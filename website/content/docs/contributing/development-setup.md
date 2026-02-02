---
title: Development Setup
weight: 3
---

Set up your local environment to contribute code to NetVantage.

## Prerequisites

- **Go 1.25+** -- [download](https://go.dev/dl/)
- **Make** -- build automation
- **Git** -- version control

## Clone and Build

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/netvantage.git
cd netvantage
make build
```

## Run Tests

```bash
make test
```

## Run Linter

```bash
make lint
```

Uses [golangci-lint](https://golangci-lint.run/) with the project's `.golangci.yml` configuration.

## Development Workflow

1. **Fork** the repository and create a branch from `main`
2. **Branch naming**: `feature/short-description`, `fix/short-description`, `refactor/short-description`
3. Make your changes following the code style guidelines
4. Write tests for new functionality
5. Run checks: `make test && make lint`
6. Commit using [Conventional Commits](/docs/contributing/#commit-conventions)
7. Push and open a Pull Request against `main`

## Code Style

| Rule | Details |
|------|---------|
| Formatting | `gofmt` (enforced by CI) |
| Linting | golangci-lint with project configuration |
| Error handling | Return errors, don't panic. Wrap with context. |
| Logging | `go.uber.org/zap` structured logging (never `fmt.Println`) |
| Context | Pass `context.Context` as the first parameter |
| Interfaces | Define in consumer package, not provider |
| Tests | Table-driven. Use `testify` for assertions. |
| SQL | Raw SQL with thin repository layer. No ORM. |
| Comments | Only where logic isn't self-evident |

## Project Structure

```
cmd/
  netvantage/     Server entry point
  scout/          Agent entry point
internal/
  config/         Configuration management
  event/          In-memory event bus
  registry/       Plugin lifecycle and dependency resolution
  server/         HTTP server
  version/        Build-time version injection
  recon/          Network discovery module
  pulse/          Monitoring module
  dispatch/       Agent management module
  vault/          Credential management module
  gateway/        Remote access module
  scout/          Agent core logic
pkg/
  plugin/         Public plugin SDK (Apache 2.0)
  models/         Shared data types
api/
  proto/v1/       gRPC service definitions
docs/
  adr/            Architecture Decision Records
  guides/         Developer guides
  requirements/   Requirement specifications
web/              React dashboard (TypeScript)
```

## Pull Request Process

1. Fill out the [PR template](https://github.com/HerbHall/netvantage/blob/main/.github/pull_request_template.md) completely
2. Ensure CI passes (build, test, lint, license check)
3. Request review from a maintainer
4. Address review feedback
5. Squash or rebase as needed before merge

For more details, see the full [CONTRIBUTING.md](https://github.com/HerbHall/netvantage/blob/main/CONTRIBUTING.md) in the repository.
