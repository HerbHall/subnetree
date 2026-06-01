---
phase: delivery
updated: 2026-06-01T22:00:00Z
updated_by: claude-code
---

# SubNetree -- Current State

## Phase

Post-PR cleanup. Main is at f568739 (#601). Latest release v0.6.4; v0.6.5
queued via release-please (#603). Open PRs: 7 (1 release + 6 Dependabot).
Open issues: 8.

## Pending

- #603: release-please PR for v0.6.5 (auto-managed; merge to cut the release)
- #493-#498: deployment validation phases (human hardware work)
- #487: community engagement launch prep
- #499: content capture for community launch
- Dependabot backlog: #588 (zap), #591 (x/mod), #592 (sqlite), #596 (grpc),
  #599 (x/net 0.55.0), #600 (go-sdk). #591/#599 have stale bases after the
  #602 x/* bump and need a rebase; batch with `@dependabot squash and merge`.

## Next Actions

- Merge #603 to release v0.6.5 (carries the #601/#602 fixes)
- Clear the Dependabot backlog (rebase #591/#599 first; AP#200 batch-merge)
- #493-#498: Docker Desktop / UNRAID / Proxmox validation
- #499: content capture for community launch
- #286: IPv6 scanning (Phase 4+)

## Recently Completed

Session 2026-06-01 (2 PRs merged):

- Merged PR #601 (f568739): Fixed device-table IP sort -- IPs were sorted
  lexicographically (192.168.1.111 before 192.168.1.12). New pure
  `compareIpAddresses` comparator in `web/src/lib/ip.ts` (octet-numeric,
  valid-IPv4-first fallback), 11 tests. Closes #585. Copilot review caught a
  real bug: `localeCompare` fallback was locale-dependent, replaced with
  code-unit comparison (captured as SYN#6585).
- Merged PR #602 (security): Cleared a day-of govulncheck DB hit (10 vulns)
  blocking the whole PR queue -- x/crypto 0.50.0->0.52.0, x/net 0.52.0->0.54.0,
  Go directive 1.25.9->1.25.10 (stdlib CVEs), Dockerfile go-builder
  1.25.9->1.25.10-alpine sibling update. Superseded Dependabot #598 (auto-closed).
- Dashboard skill (`.claude/skills/dashboard/`) de-hardcoded: stale absolute
  paths (family-folder move + Windows profile-casing + deleted dev-mode skill)
  replaced with relative paths + bare git + ~/.claude tilde. Local-only
  (.claude is gitignored). Captured as SYN#6582; DevKit issue #389 filed.

Session 2026-04-23 (3 PRs merged, 6 Dependabot alerts cleared):

- Merged PR #580 (7f41f77): Ansible dynamic inventory plugin + YAML export
  endpoint. Closes #489. Go toolchain 1.25.8->1.25.9, Dockerfile sibling update.
- Merged PR #581 (47a4729): Resolved 6 Dependabot alerts in web deps.
- Merged PR #583 (2113239): Closes #582. Unpinned recharts +
  eslint-plugin-react-hooks, refactored revealed anti-patterns.

## Queued (Roadmap)

- #280: multi-tenancy support (Phase 4+)
- #286: IPv6 scanning (Phase 4+)
- #289: PostgreSQL + TimescaleDB (Phase 4+)

## Start Here (Cold Start Protocol)

1. Read this file
2. Call `samverk get_digest --since 168h` if MCP is configured
3. Read open issues if relevant to the task
4. Proceed -- do not ask the user to explain project state
