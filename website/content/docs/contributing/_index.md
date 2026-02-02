---
title: Contributing
weight: 3
---

NetVantage welcomes contributions from the community. Whether you're reporting a bug, requesting a feature, or submitting code, this section covers everything you need to know.

## Quick Links

{{< cards >}}
  {{< card link="reporting-bugs" title="Report a Bug" icon="exclamation-circle" subtitle="Found something broken? Help us fix it." >}}
  {{< card link="feature-requests" title="Request a Feature" icon="light-bulb" subtitle="Have an idea? We'd love to hear it." >}}
  {{< card link="development-setup" title="Development Setup" icon="code" subtitle="Set up your environment to contribute code." >}}
{{< /cards >}}

## Contributor License Agreement

All contributors must sign the [CLA](https://github.com/HerbHall/netvantage/blob/main/.github/CLA.md) before their first PR can be merged. This is automated -- the CLA bot will prompt you on your first pull request.

**Why a CLA?** NetVantage uses a split licensing model (BSL 1.1 + Apache 2.0). The CLA ensures a clean IP chain, which is essential for the project's long-term sustainability.

## Code of Conduct

All contributors are expected to follow our [Code of Conduct](https://github.com/HerbHall/netvantage/blob/main/CODE_OF_CONDUCT.md). We are committed to providing a welcoming and inclusive environment.

## Commit Conventions

NetVantage uses [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add device type icon mapping
fix: prevent duplicate scan entries on network change
refactor: extract plugin lifecycle into separate package
docs: update API endpoint documentation
test: add integration tests for Recon module
chore: update golangci-lint to v1.62
```

- Keep the subject line under 72 characters
- Use the imperative mood ("add", not "added" or "adds")
- Reference issues when applicable: `fix: resolve scan timeout (#42)`

## Licensing

- Contributions to **core code** fall under BSL 1.1 (covered by the CLA)
- Contributions to the **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`, `api/proto/`) fall under Apache 2.0
- Do not introduce dependencies with GPL, AGPL, LGPL, or SSPL licenses
