# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the NetVantage project. ADRs document significant architectural decisions, their context, and rationale.

## Format

We use [MADR](https://adr.github.io/madr/) (Markdown Any Decision Records) -- a lightweight, GitHub-renderable format.

## When to Write an ADR

- Technology choice affecting multiple modules
- Architectural pattern adoption
- Security-sensitive design decisions
- Breaking changes to APIs or protocols
- Any decision a future contributor would ask "why?"

## Index

| # | Title | Status | Date |
|---|-------|--------|------|
| 0001 | [Split Licensing Model](0001-split-licensing-model.md) | Accepted | 2025-01 |
| 0002 | [SQLite-First Database Strategy](0002-sqlite-first-database.md) | Accepted | 2025-01 |
| 0003 | [Plugin Architecture (Caddy Model)](0003-plugin-architecture-caddy-model.md) | Accepted | 2025-01 |
| 0004 | [Integer-Based Protocol Versioning](0004-integer-protocol-versioning.md) | Accepted | 2025-01 |

## Lifecycle

- **Proposed** -- Under discussion, not yet decided
- **Accepted** -- Decision made, implementation may be pending
- **Deprecated** -- No longer relevant (superseded or abandoned)
- **Superseded** -- Replaced by a newer ADR (link to replacement)

## Creating a New ADR

1. Copy [template.md](template.md)
2. Name it `NNNN-short-title.md` (next sequential number)
3. Fill in all sections
4. Submit via PR for review
5. Update this index
