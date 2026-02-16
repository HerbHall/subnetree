# 8. Consolidate workspace -- absorb coordination and research

Date: 2026-02-16

## Status

Accepted

## Context

SubNetree development used three separate git repos in one VS Code workspace:
SubNetree (GitHub), .coordination (local git), and research/HomeLab (local git).
This created unnecessary complexity for a solo developer -- three repos to manage,
absolute paths in documentation, and Claude Code needing to understand multi-repo layout.

The Runbooks project demonstrated that project-local `.coordination/` (gitignored)
is simpler and sufficient for a single-project coordination pattern.

## Decision

Move `.coordination/` and `research/HomeLab/` inside `SubNetree/` as gitignored
directories. The workspace file becomes single-root. Research outputs are published
to `docs/research/` (committed) when ready for public consumption.

## Consequences

- Single project directory, single workspace root, single mental model
- Claude Code sees everything in one tree
- Two fewer local git repos to manage
- Clear publishing pipeline: research/ (private) -> docs/research/ (public)
- Path references in CLAUDE.md, settings.json, and DASHBOARD.md updated to relative
- No impact on Go code, CI/CD, or public repo structure
