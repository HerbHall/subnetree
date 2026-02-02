# ADR-0001: Split Licensing Model

## Status

Accepted

## Date

2025-01-01

## Context

NetVantage needs a licensing model that:
- Keeps the product free for personal, home-lab, and educational use
- Enables commercial monetization for business/MSP/enterprise customers
- Allows third-party plugin development without license contamination
- Maintains a clean IP chain suitable for acquisition
- Prevents competitors from forking and competing with an identical commercial product

Pure open-source licenses (MIT, Apache 2.0) offer no commercial protection. Proprietary licenses prevent community adoption. GPL/AGPL licenses deter enterprise adoption and plugin development.

## Decision

Use a split licensing model:

- **Core** (server, agent, built-in modules): Business Source License 1.1 (BSL 1.1)
  - Free for personal, home-lab, educational, and non-competing production use
  - Change Date: 4 years from each release (converts to Apache 2.0)
  - Additional Use Grant: personal/home/educational/non-competing production use
- **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`, `api/proto/`): Apache License 2.0
  - No restrictions on plugin development or distribution
  - Enables commercial and open-source plugin ecosystem

Require a Contributor License Agreement (CLA) for all contributions to maintain IP chain integrity.

## Consequences

### Positive

- Personal/home users get the full product for free, forever
- Commercial revenue path without feature-gating the core product
- Plugin developers face zero licensing friction (Apache 2.0)
- Clean IP chain for acquisition (CLA + BSL 1.1 + Apache 2.0)
- Code automatically becomes fully open source after 4 years
- Precedent set by successful projects: HashiCorp, MariaDB, CockroachDB, Sentry

### Negative

- BSL 1.1 is not OSI-approved; some open-source purists will object
- CLA creates friction for first-time contributors
- "Non-competing production use" requires clear definition to avoid ambiguity
- Some package managers and distributions may not include BSL-licensed software

### Neutral

- GPL, AGPL, LGPL, and SSPL dependencies are blocked to avoid license conflicts
- `make license-check` enforces dependency compliance in CI

## Alternatives Considered

### Alternative 1: Pure Apache 2.0

Maximum community adoption but zero commercial protection. Any company could fork and compete. Not suitable for a product intended for acquisition.

### Alternative 2: AGPL

Strong copyleft would require plugin developers to open-source their plugins. This kills the commercial plugin ecosystem and deters enterprise adoption.

### Alternative 3: Dual License (GPL + Commercial)

Requires selling commercial licenses for any proprietary use. More complex to administer and creates a hard paywall for small businesses.

### Alternative 4: SSPL (Server Side Public License)

Stronger protection than BSL but controversial and rejected by most Linux distributions. MongoDB's SSPL has faced significant community pushback.
