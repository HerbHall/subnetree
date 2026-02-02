## Documentation Requirements

### User-Facing Documentation

| Document | Description | Phase |
|----------|-------------|-------|
| README.md | Quick start, feature overview, screenshots | 1 |
| Installation Guide | Single binary, Docker, Docker Compose | 1 |
| Configuration Reference | All YAML keys, env vars, defaults | 1 |
| User Guide | Dashboard walkthrough, common workflows | 1 |
| Admin Guide | User management, backup/restore, upgrades | 2 |
| API Reference | OpenAPI 3.0 spec, auto-generated | 1 |
| Agent Deployment Guide | Windows, Linux, macOS installation | 1b/2/3 |

### Developer Documentation

| Document | Description | Phase |
|----------|-------------|-------|
| Architecture Overview | System design, plugin system, data flow | 1 |
| Plugin Developer Guide | Creating custom modules, role interfaces, SDK | 2 |
| Contributing Guide (CONTRIBUTING.md) | Development setup, code style, PR process, testing, CLA | 1 |
| Plugin API Changelog | Breaking changes by API version | 2 |
| Example Plugins | Webhook notifier, Prometheus exporter, alternative credential store | 2 |

### Community Health Files (GitHub)

Standard files that GitHub recognizes and surfaces in the repository UI. These establish project professionalism and reduce friction for contributors and evaluators.

| File | Description | Phase |
|------|-------------|-------|
| `CONTRIBUTING.md` | Development setup, PR process, code style, testing expectations, CLA | 1 |
| `SECURITY.md` | Vulnerability disclosure process, supported versions, security contacts | 1 |
| `.github/pull_request_template.md` | PR checklist: description, tests, lint, breaking changes | 1 |
| `.github/ISSUE_TEMPLATE/bug_report.md` | Bug report template (exists) | 1 |
| `.github/ISSUE_TEMPLATE/feature_request.md` | Feature request template (exists) | 1 |
| `.github/FUNDING.yml` | Sponsor button configuration (exists) | 1 |
| `.github/workflows/ci.yml` | CI pipeline: lint, test, build, license check | 1 |
| `.github/workflows/release.yml` | Release pipeline: GoReleaser, Docker, SBOM, signing | 1 |
| `CODE_OF_CONDUCT.md` | Contributor Covenant or similar code of conduct | 1 |
| `SUPPORTERS.md` | Financial supporter recognition (exists) | 1 |
| `LICENSING.md` | Human-readable licensing explanation (exists) | 1 |

### README Structure (Target State)

The README is the project's front door. It must convert a skeptical visitor into someone who tries the software. Target structure for Phase 1:

```
# NetVantage
[badges: CI | Go version | License | Latest Release | Docker Pulls]

One-sentence description + key screenshot/GIF

## Why NetVantage?
- Feature comparison table (vs Zabbix, LibreNMS, Uptime Kuma, Domotz)
- "One tool instead of five" value proposition

## Current Status
- What works today (honest)
- What's in progress
- Link to roadmap

## Quick Start
### Binary
### Docker (one-liner)
### Docker Compose

## Screenshots
Dashboard, topology map, device detail, scan in progress

## Architecture
[existing architecture diagram]

## Modules
[existing module table]

## Development
Build, test, lint commands

## Contributing
Link to CONTRIBUTING.md

## Support the Project
Sponsor links

## License
Clear BSL 1.1 explanation with "free for personal, home-lab, and non-competing production use"
```
