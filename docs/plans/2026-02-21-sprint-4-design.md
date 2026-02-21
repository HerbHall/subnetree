# Sprint 4: Multi-Agent Sprint Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:dispatching-parallel-agents to execute this plan wave-by-wave.

**Goal:** Ship MCP server interface, Scout hardware collection improvements, Windows installer, and updated docs in a two-wave parallel agent sprint.

**Architecture:** Wave-based parallel execution with 3 code agents + 1 background research in Wave 1, followed by 1 docs agent in Wave 2. Zero file overlap between Wave 1 agents. Wave 2 documents what Wave 1 shipped.

**Tech Stack:** Go 1.25+, MCP Go SDK, Proxmox VE API, Inno Setup 6, MkDocs Material

---

## Wave Structure

```text
Wave 1 (parallel, zero file overlap):
  Agent A: #439 MCP server interface       [internal/mcp/, main.go, go.mod]
  Agent B: #438 Scout hardware + Proxmox   [internal/scout/profiler/, internal/recon/]
  Agent C: #431 Inno Setup installer       [installer/, .github/workflows/]
  Agent BG: Competitor re-scan             [.coordination/ only, background]

Wave 2 (after Wave 1 merges):
  Agent D: #430 + #440 Docs                [docs/]
```

## Shared File Conflict Matrix

| File | Agent A | Agent B | Agent C | Agent D |
|------|---------|---------|---------|---------|
| `cmd/subnetree/main.go` | WRITE | - | - | - |
| `go.mod` / `go.sum` | WRITE | - | - | - |
| `internal/mcp/` | CREATE | - | - | - |
| `internal/scout/profiler/` | - | WRITE | - | - |
| `internal/recon/` | - | WRITE | - | - |
| `installer/` | - | - | CREATE | - |
| `.github/workflows/` | - | - | WRITE | - |
| `docs/` | - | - | - | WRITE |

**Verdict:** Zero overlap in Wave 1. Safe for full parallelism.

---

## Agent A: MCP Server Interface (#439)

**Branch:** `feature/issue-439-mcp-server`

**Context for agent:** SubNetree is a modular Go network monitoring platform. Each module implements `plugin.Plugin` (see `pkg/plugin/plugin.go`). Modules register in `cmd/subnetree/main.go` at line 152-164. The MCP (Model Context Protocol) server lets external AI tools query device inventory. Research finding RF-012 recommends the official Go SDK at `github.com/modelcontextprotocol/go-sdk`.

### Files to Create

- `internal/mcp/module.go` -- plugin.Plugin implementation
- `internal/mcp/module_test.go` -- contract tests + tool tests
- `internal/mcp/tools.go` -- MCP tool definitions and handlers

### Files to Modify

- `cmd/subnetree/main.go` -- add import + register `mcp.New()` in modules slice (line ~163)
- `go.mod` -- add MCP SDK dependency

### Implementation Steps

1. **Create `internal/mcp/module.go`**
   - Implement `plugin.Plugin` interface: `Info()`, `Init()`, `Start()`, `Stop()`
   - PluginInfo: name "mcp", role `roles.RoleMCP` (or a new const), dependencies: `["recon"]`
   - In `Init()`: receive store and event bus via `plugin.Dependencies`
   - In `Start()`: initialize MCP server with SDK, register tools
   - Define consumer-side interfaces for querying devices/hardware (avoid importing internal/recon directly):

   ```go
   type DeviceQuerier interface {
       GetDevice(ctx context.Context, id string) (*models.Device, error)
       ListDevices(ctx context.Context, limit, offset int) ([]models.Device, int, error)
       GetDeviceHardware(ctx context.Context, deviceID string) (*models.DeviceHardware, error)
       GetHardwareSummary(ctx context.Context) (*models.HardwareSummary, error)
       QueryDevicesByHardware(ctx context.Context, query models.HardwareQuery) ([]models.Device, int, error)
   }
   ```

2. **Create `internal/mcp/tools.go`**
   - Register 7 tools with the MCP SDK:
     - `get_device` -- single device by ID
     - `list_devices` -- paginated device list
     - `get_hardware_profile` -- full hardware profile for a device
     - `query_devices` -- filter by hardware criteria (RAM, CPU, OS, etc.)
     - `get_fleet_summary` -- aggregate stats (total devices, by type, by OS)
     - `get_service_inventory` -- all detected services across fleet
     - `get_stale_devices` -- devices not seen within threshold
   - Each tool: validate input, call DeviceQuerier, return JSON result
   - NEVER expose vault credentials through any tool
   - Log every tool call to the audit system (use event bus: `mcp.tool.called`)

3. **HTTP transport setup**
   - Mount MCP handler at `/api/v1/mcp` via the plugin's `Routes()` method (HTTPProvider interface)
   - Localhost-only by default (configurable via `mcp.listen_address`)
   - API key auth: read from config `mcp.api_key`, validate in middleware

4. **Register in main.go**
   - Add import: `"github.com/HerbHall/subnetree/internal/mcp"`
   - Add to modules slice: `mcp.New(),` (after `autodoc.New()`, line ~163)

5. **Add `subnetree mcp` CLI subcommand** (in main.go subcommand dispatch, line ~57)
   - Runs MCP server in stdio mode for Claude Desktop integration
   - Reuses same tool definitions but with stdio transport instead of HTTP

### Testing

- `plugintest.TestPluginContract(t, mcp.New())` -- standard contract test
- Table-driven tests for each tool handler with mock DeviceQuerier
- Test API key middleware (valid key, invalid key, missing key)
- Test that vault-related queries are rejected

### CI Checklist

```bash
go build ./...
go test ./internal/mcp/...
GOOS=linux GOARCH=amd64 go build ./...
swag init -g cmd/subnetree/main.go -o api/swagger --parseDependency --parseInternal
```

---

## Agent B: Scout Hardware + Proxmox (#438)

**Branch:** `feature/issue-438-hardware-collection`

**Context for agent:** SubNetree's Scout agent collects hardware profiles on monitored devices. Linux hardware collection is already implemented in `internal/scout/profiler/hardware_other.go` (CPU, RAM, disk, NIC, DMI). However, GPU collection is missing on Linux (Windows has it via WMI in `hardware_windows.go:70-83`). The hardware profile data flows: Scout -> gRPC ReportProfile -> dispatch event -> recon hardware_bridge.go -> normalized tables. Issue #438 also needs an on-demand refresh endpoint and Proxmox API integration.

### Files to Create

- `internal/recon/proxmox_collector.go` -- Proxmox VE API client for agentless hardware collection
- `internal/recon/proxmox_collector_test.go` -- tests with httptest mock server

### Files to Modify

- `internal/scout/profiler/hardware_other.go` -- add Linux GPU collection via `/sys/class/drm/` and `lspci`
- `internal/recon/hardware_handlers.go` -- add `POST /devices/{id}/hardware/refresh` endpoint
- `internal/recon/recon.go` -- register refresh route + Proxmox collector initialization

### Implementation Steps

1. **Add Linux GPU collection to `hardware_other.go`**
   - Read GPU info from `/sys/class/drm/card*/device/` (vendor, device files)
   - Fallback: parse `lspci -mm` output for VGA/3D controllers
   - Read VRAM from `/sys/class/drm/card*/device/mem_info_vram_total` (AMD) or `/proc/driver/nvidia/gpus/*/information` (NVIDIA)
   - Populate `hw.Gpus` field (same proto `GPUInfo` message that Windows uses)
   - Follow existing pattern: best-effort, log debug on failure, never fail the full collection

2. **Add on-demand refresh endpoint**
   - Handler in `hardware_handlers.go`:

   ```go
   // POST /api/v1/recon/devices/{id}/hardware/refresh
   // Triggers a hardware profile re-collection for a device.
   // For agent-managed devices: publishes event requesting Scout to re-report.
   // For agentless devices: triggers direct collection (Proxmox, etc.)
   ```

   - For agent-managed devices: publish `dispatch.device.refresh_hardware` event with device ID
   - For agentless/manual devices: return 200 with status "refresh requested"
   - Register route in `recon.go` Routes() method
   - Add swagger annotations

3. **Proxmox VE API collector**
   - `proxmox_collector.go`: HTTP client for Proxmox VE REST API
   - Auth via API token (stored in Vault, retrieved via consumer-side `CredentialProvider` interface)
   - Endpoints to query:
     - `GET /api2/json/nodes` -- list cluster nodes
     - `GET /api2/json/nodes/{node}/status` -- node hardware (CPU, RAM, uptime)
     - `GET /api2/json/nodes/{node}/disks/list` -- storage devices
     - `GET /api2/json/nodes/{node}/qemu` -- VMs
     - `GET /api2/json/nodes/{node}/lxc` -- LXC containers
   - Map Proxmox data to `models.DeviceHardware`, `models.DeviceStorage`, `models.DeviceService`
   - Set `collection_source = "proxmox-api"` on all rows
   - Configuration: `recon.proxmox.url`, `recon.proxmox.credential_id` (references Vault)

4. **Register Proxmox in recon module**
   - Add Proxmox config section to recon module Init()
   - Optional: only initialize if `recon.proxmox.url` is configured
   - Wire into scan pipeline as an optional phase (after network scan)

### Testing

- GPU collection: unit test `parseLspciOutput()` with sample lspci data
- Refresh endpoint: httptest with mock store, verify event published
- Proxmox collector: httptest mock server returning sample Proxmox API responses
- Table-driven tests for Proxmox node/VM/LXC data mapping

### CI Checklist

```bash
go build ./...
go test ./internal/recon/... ./internal/scout/profiler/...
GOOS=linux GOARCH=amd64 go build ./...
swag init -g cmd/subnetree/main.go -o api/swagger --parseDependency --parseInternal
```

---

## Agent C: Inno Setup Windows Installer (#431)

**Branch:** `feature/issue-431-inno-setup`

**Context for agent:** SubNetree's Scout agent is a single Go binary (`scout.exe` on Windows). Currently distributed as a bare binary via GoReleaser. Users manually place it and register it. Issue #431 adds a proper Windows installer using Inno Setup 6 that bundles the Scout binary, creates Start Menu entries, and optionally registers it as a service (future #428).

### Files to Create

- `installer/inno/subnetree-scout.iss` -- Inno Setup script
- `installer/inno/README.md` -- build instructions
- `.github/workflows/build-installer.yml` -- CI workflow for building the installer

### Files to Modify

- `.github/workflows/release.yml` -- add installer artifact upload step (or reference new workflow)

### Implementation Steps

1. **Create Inno Setup script** (`installer/inno/subnetree-scout.iss`)
   - App name: "SubNetree Scout Agent"
   - Default install dir: `{autopf}\SubNetree\Scout`
   - Install the `scout.exe` binary (expects it at `installer/inno/bin/scout.exe` -- CI copies it)
   - Create Start Menu group: "SubNetree" with "Scout Agent" shortcut
   - Create uninstaller entry in Windows Add/Remove Programs
   - Add `{app}` to system PATH (optional, checkbox)
   - License page: display BSL 1.1 license text
   - Version info: read from environment variable `SCOUT_VERSION` (injected by CI)
   - Output: `SubNetreeScout-{version}-setup.exe`
   - Minimum Windows version: Windows 10

2. **Create CI workflow** (`.github/workflows/build-installer.yml`)
   - Trigger: on release publish (same as main release)
   - Steps:
     - Download Scout Windows binary from release assets
     - Install Inno Setup via `jrsoftware/iscc` GitHub Action or choco
     - Run `iscc installer/inno/subnetree-scout.iss`
     - Upload `SubNetreeScout-*-setup.exe` to the release

3. **Update release workflow**
   - Add a step or job that triggers the installer build after Scout binary is uploaded
   - OR: integrate directly into the existing Windows build job

4. **Create README** (`installer/inno/README.md`)
   - Prerequisites: Inno Setup 6 on Windows
   - Local build: `iscc subnetree-scout.iss`
   - How CI builds it automatically

### Testing

- Manual: build installer locally with Inno Setup 6, verify install/uninstall on Windows
- CI: verify workflow runs and produces .exe artifact
- Verify uninstall removes all files and registry entries

### CI Checklist

- Verify the `.iss` script compiles with `iscc` (CI step)
- Verify installer artifact is uploaded to release

---

## Agent BG: Competitor Re-Scan (Background Research)

**Branch:** N/A (research only, outputs to gitignored .coordination/)

**Context for agent:** Monthly competitor landscape scan. Check GitHub stats for Scanopy, NetAlertX, Beszel, and any new entrants in the homelab network monitoring space. Previous analysis: RF-001 (Scanopy), RF-006 (NetAlertX, Beszel), RF-010 (Beszel agent architecture).

### Research Tasks

1. **Check competitor GitHub stats** (use `gh api`):
   - `scanopy/scanopy` -- stars, recent releases, open issues, recent commits
   - `jokob-sk/NetAlertX` -- stars, recent releases, open issues
   - `henrygd/beszel` -- stars, recent releases (was 19.4k in Feb)
   - `aceberg/WatchYourLAN` -- stars, activity level

2. **Scan for new entrants**:
   - GitHub search: `homelab network discovery self-hosted` sorted by stars
   - GitHub search: `network topology visualization self-hosted` sorted by recent
   - Check awesome-selfhosted for any new entries in Monitoring/CMDB categories

3. **Check high-value issues/discussions** on competitors:
   - Top issues by reactions on Scanopy (reveals user demand)
   - Recent closed issues on Beszel (reveals development velocity)

4. **Output**: Write findings to `.coordination/research-findings.md` as RF-013

### Deliverable Format

```markdown
### RF-013 - Monthly Competitor Re-Scan (February 2026)

- **Source**: gh API analysis of 4 competitors + GitHub search for new entrants
- **Impact**: [Low/Medium/High]
- **Summary**: [Stats delta since last scan, new entrants, notable developments]
- **Action**: [Positioning adjustments if any]
```

---

## Agent D: Documentation Update (#430 + #440)

**Branch:** `feature/issue-430-440-docs`

**Wave 2 -- runs after Wave 1 merges.**

**Context for agent:** SubNetree just shipped hardware asset profiles (PRs #441-#443) and MCP server interface (#439, Wave 1). This agent updates requirements docs, writes ADRs, and creates the Scout uninstall guide. The three-tier doc model: README = landing page, docs site (MkDocs) = user guides, in-repo docs/ = contributor docs.

### Files to Create

- `docs/guides/scout-uninstall.md` -- platform-specific Scout removal guide (#430)
- `docs/adr/0009-hardware-asset-profiles.md` -- ADR for hardware profile design (#440)
- `docs/adr/0010-mcp-server-architecture.md` -- ADR for MCP server decisions (#440)

### Files to Modify

- `docs/requirements/10-data-model.md` -- add hardware profile tables (device_hardware, device_storage, device_gpu, device_services)
- `docs/requirements/02-architecture-overview.md` -- add MCP module to module table
- `docs/requirements/13-dashboard-architecture.md` -- add hardware tab to page list
- `docs/requirements/08-scout-agent.md` -- update with GPU collection, Proxmox integration notes

### Implementation Steps

1. **Scout uninstall guide** (`docs/guides/scout-uninstall.md`) (#430)
   - Sections: Linux (systemd), Linux (OpenRC), Linux (Docker), Windows (manual), Windows (installer)
   - Each section: stop service, remove binary, remove config, remove data, clean up system entries
   - Include verification steps ("confirm Scout is fully removed")

2. **ADR-0009: Hardware Asset Profiles** (#440)
   - Context: need structured hardware inventory beyond basic discovery
   - Decision: 4 normalized tables (device_hardware, device_storage, device_gpu, device_services)
   - Alternatives considered: JSON column (rejected: poor queryability), single wide table (rejected: 1:many relationships)
   - Consequences: migration v10, manual override support, collection_source provenance

3. **ADR-0010: MCP Server Architecture** (#440)
   - Context: external AI tools need structured access to device inventory
   - Decision: Official Go MCP SDK, HTTP transport at `/api/v1/mcp`, consumer-side interfaces
   - Alternatives considered: Custom REST API (rejected: no MCP ecosystem benefits), stdio-only (rejected: can't embed in server)
   - Consequences: new dependency, localhost-only default, audit logging

4. **Update requirement files**
   - `10-data-model.md`: add hardware profile entity descriptions and table schemas
   - `02-architecture-overview.md`: add MCP row to module table
   - `13-dashboard-architecture.md`: add hardware tab to device detail page list
   - `08-scout-agent.md`: note GPU collection on Linux, Proxmox integration

### Testing

- No code tests -- verify markdown passes `markdownlint`
- Verify ADR numbering is sequential (0008 was last)
- Verify cross-references resolve

### CI Checklist

- Markdownlint passes on all new/modified docs
- No broken internal links

---

## Execution Sequence

### Pre-Sprint

1. Ensure `main` is clean: `git status` (verified during sync audit)
2. Create all 4 branches from main:
   ```bash
   git checkout -b feature/issue-439-mcp-server main
   git checkout -b feature/issue-438-hardware-collection main
   git checkout -b feature/issue-431-inno-setup main
   git checkout -b feature/issue-430-440-docs main
   git checkout main
   ```

### Wave 1 Execution

3. Launch Agents A, B, C in parallel (foreground)
4. Launch Agent BG (background research)
5. Wait for all agents to complete
6. Sort changes into correct branches via `git stash`/`git checkout`/`git stash pop`
7. For each branch: verify, commit, push, create PR
8. Monitor CI, fix any failures
9. Merge PRs sequentially: A -> B -> C (rebase between each if needed)
10. Process Agent BG research findings

### Wave 2 Execution

11. `git fetch origin && git checkout feature/issue-430-440-docs && git rebase origin/main`
12. Launch Agent D
13. Verify, commit, push, create PR
14. Merge after CI passes

### Post-Sprint

15. Close issues: #439, #438, #431, #430, #440
16. Update `.coordination/status.md` and `priorities.md`
17. Update MEMORY.md with sprint results

---

## Git Safety (Embedded in All Agent Prompts)

```text
## Git Safety -- IMPORTANT

Do NOT commit, push, or create PRs. Leave all changes unstaged in the working tree.
The main context handles all git operations (commit, push, PR creation).
If you run `git checkout`, you will destroy other parallel agents' unstaged changes.
```

## Go Agent CI Checklist (Embedded in Agents A, B)

```text
## Pre-Completion CI Checklist (MUST verify before finishing)

1. `go build ./...` -- Compilation
2. `go test ./...` -- Tests (skip -race on Windows MSYS)
3. `GOOS=linux GOARCH=amd64 go build ./...` -- Cross-compile check
4. If HTTP handlers added: `swag init -g cmd/subnetree/main.go -o api/swagger --parseDependency --parseInternal`

Watch for: gosec G101, gocritic (unnamedResult, appendCombine, rangeValCopy, paramTypeCombine, httpNoBody), bodyclose, build-tag file visibility.
```
