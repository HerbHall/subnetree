---
phase: delivery
updated: 2026-04-23T17:40:00Z
updated_by: claude-code
---

# SubNetree -- Current State

## Phase

Post-PR cleanup. Main is at 2113239. Zero open PRs. Open Dependabot alerts: 0.

## Pending

- #493-#498: deployment validation phases (human hardware work)
- #487: community engagement launch prep
- #499: content capture for community launch

## Next Actions

- #493-#498: Docker Desktop / UNRAID / Proxmox validation
- #499: content capture for community launch
- #286: IPv6 scanning (Phase 4+)

## Recently Completed

Session 2026-04-23 (3 PRs merged, 6 Dependabot alerts cleared):

- Merged PR #580 (7f41f77): Ansible dynamic inventory plugin + YAML export endpoint.
  Closes #489. Fixes along the way: gofmt on `internal/recon/ansible.go`, Go toolchain
  bump 1.25.8 -> 1.25.9 (stdlib CVE hotfix: GO-2026-4947/4946/4870/4869/4865), Dockerfile
  golang:1.25.8-alpine -> 1.25.9-alpine sibling update.
- Merged PR #581 (47a4729): Resolved 6 Dependabot alerts in web deps. happy-dom bump +
  pnpm.overrides for minimatch/picomatch/flatted. Scoped PR kept narrow via exact pins on
  recharts (3.7.0) and eslint-plugin-react-hooks (7.0.1); follow-up filed as #582.
- Merged PR #583 (2113239): Closes #582. Unpinned both deps (eslint-plugin-react-hooks
  ^7.1.1, recharts ^3.8.1) and refactored the code each bump revealed -- derived-state
  effect in color-picker.tsx replaced with "adjust state on prop change" pattern;
  setup.tsx network-interfaces fetch migrated to TanStack Query; two Recharts Tooltip
  sites moved to JSX element form with Partial<TooltipContentProps<...>> per KG#6.
- Claude Code plugin registry repaired (Windows profile case-drift; marketplaces
  remove+re-add). Captured as SYN#5954 + devkit#274.
- Autolearn: 4 new Synapset memories this session (#5954 plugin gotcha, #5956 pnpm
  override requires clean install, #5957 caret-bump cascade needs exact-pin in scoped
  PRs, #5958 Go toolchain Dockerfile sibling-update). DevKit issues filed: #274, #275.

## Queued (Roadmap)

- #280: multi-tenancy support (Phase 4+)
- #286: IPv6 scanning (Phase 4+)
- #289: PostgreSQL + TimescaleDB (Phase 4+)

## Start Here (Cold Start Protocol)

1. Read this file
2. Call `samverk get_digest --since 168h` if MCP is configured
3. Read open issues if relevant to the task
4. Proceed -- do not ask the user to explain project state
