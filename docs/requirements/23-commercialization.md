## Commercialization Strategy

### Strategic Intent

**Free for personal and home use forever.** This is a firm commitment, not a marketing tactic. HomeLab enthusiasts, students, and hobbyists will always have full access to every feature at no cost. This community is the foundation of adoption, feedback, and evangelism.

**Built for acquisition, not subscription revenue.** The primary goal is to build an excellent product with a passionate community, then position it for acquisition by a larger platform company. The codebase, documentation, community, and clean IP chain are the product -- not just the software.

The founder is not planning to build or operate a subscription business. However, the commercial licensing structure and pricing tiers are maintained and ready for a future owner who may want to monetize that way. This keeps options open for acquirers without burdening the founder with sales/support infrastructure.

**Commercial licensing is an option, not the goal.** The BSL 1.1 license and tiered pricing structure exist to:

1. Protect the project from competitors offering it as a hosted service
2. Make the project attractive to acquirers who want a revenue path
3. Provide a clear commercial option if the founder's plans change

### Licensing & Intellectual Property

#### Split Licensing Model

| Component | License | Rationale |
|-----------|---------|-----------|
| **Core Server + Scout Agent** | BSL 1.1 (Business Source License) | Protects commercial rights; prevents competing hosted offerings; acquirer-friendly (HashiCorp/IBM precedent) |
| **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`) | Apache 2.0 | Maximizes plugin ecosystem adoption; no friction for community or commercial plugin authors |
| **Protobuf Definitions** (`api/proto/`) | Apache 2.0 | Allows third-party agents and integrations |
| **Community Plugins** (`plugins/community/`) | Apache 2.0 (recommended default) | Contributors choose; Apache 2.0 template provided |

#### BSL 1.1 Terms (Core)

- **Change Date:** 4 years from each release date
- **Change License:** Apache 2.0 (code auto-converts after Change Date)
- **Additional Use Grant:** Non-competing production use permitted. Personal, HomeLab, and educational use always permitted regardless of this grant.
- **Commercial Use:** Requires a paid license from the copyright holder for:
  - Offering SubNetree as a hosted/managed service
  - Embedding SubNetree in a commercial product that competes with SubNetree offerings
  - Reselling or white-labeling SubNetree

#### Contributor License Agreement (CLA)

- **Required** for all contributions via CLA Assistant (GitHub App)
- Contributors sign once via GitHub comment on their first PR
- Grants the project owner:
  - Copyright assignment or broad license grant to contributions
  - Right to relicense contributions under any terms
  - Patent license for contributions
- **Essential for acquisition:** Clean IP ownership chain required by acquirers

#### Trademark

- Use **SubNetree™** (common-law TM symbol) immediately to establish rights
- Defer USPTO registration until closer to commercialization
- Trademark policy: forks may not use the "SubNetree" name
- Trademark guidelines documented in TRADEMARK.md

#### Dependency Compliance

- `go-licenses` integrated into CI pipeline
- Block any dependency with GPL, AGPL, LGPL, or SSPL license (incompatible with BSL 1.1)
- Allowed: MIT, BSD-2, BSD-3, Apache 2.0, ISC, MPL-2.0 (file-level copyleft only)
- License audit report generated on every build
- **Dual-licensed packages:** `eclipse/paho.mqtt.golang` -- elect EDL-1.0 (BSD-3-Clause) option
- **Weak copyleft:** `hashicorp/go-plugin` (MPL-2.0) -- use as unmodified library only
- **Docker images:** Use only official `guacamole/guacd` (Apache 2.0); avoid `flcontainers/guacamole` (GPL v3)
- Full dependency audit completed: **zero incompatible dependencies** found across all Go and npm packages

#### Repository Licensing Structure

```
d:\SubNetree\
  LICENSE                    # BSL 1.1 (covers everything by default)
  LICENSING.md              # Human-readable explanation of the licensing model
  pkg/
    plugin/
      LICENSE               # Apache 2.0
    roles/
      LICENSE               # Apache 2.0
    models/
      LICENSE               # Apache 2.0
  api/
    proto/
      LICENSE               # Apache 2.0
  plugins/
    community/
      LICENSE               # Apache 2.0 (template)
```

### Pricing Model: Ready for Future Owner

The pricing structure below is maintained as a ready-to-use commercial model for an acquirer or future pivot -- not as an active revenue stream the founder plans to operate. All tiers have **unlimited devices and unlimited customization**. Pricing is based on team/business features, not scale or functionality.

| Tier | Price | Target | Features |
|------|-------|--------|----------|
| **Community** | Free forever | HomeLabbers, personal, educational | All modules, all plugins, unlimited devices, single user, full customization, community support |
| **Team** | $9/month | Small business, household teams | + Multi-user (up to 5), OIDC/SSO, scheduled reports, email support |
| **Professional** | $29/month | Small business IT, consultants | + RBAC, audit logging, API access, priority support |

**Principle:** The free tier is never crippled. It includes every module, every plugin, and every customization option. Paid tiers add collaboration and business operations features (multi-user, SSO, audit trails) that are genuinely unnecessary for a solo home user.

**Why maintain pricing if not selling?** Acquirers want to see a clear path to revenue. Having a defined, reasonable pricing structure signals that commercialization has been thought through. It also keeps the door open if circumstances change.

### Community Contributions

Free and home users are the foundation of the project. Their contributions are valued and recognized.

#### Non-Financial Contributions

| Contribution | Channel | Recognition |
|-------------|---------|-------------|
| Bug reports | GitHub Issues (templated) | Contributor credit in release notes |
| Feature requests | GitHub Discussions | Acknowledgment if implemented |
| Beta testing | Opt-in beta channel | Early access + tester recognition |
| Documentation | Pull requests | Contributor credit + CLA on file |
| Plugin development | Apache 2.0 SDK | Listed in plugin directory |
| Community support | GitHub Discussions | Community helper recognition |

#### Voluntary Financial Support

Three platforms, zero obligation. All support is voluntary and does not unlock additional features -- the free tier is always complete.

| Platform | Type | Link |
|----------|------|------|
| **GitHub Sponsors** | Recurring or one-time | github.com/sponsors/HerbHall |
| **Ko-fi** | Recurring or one-time | ko-fi.com/herbhall |
| **Buy Me a Coffee** | One-time or membership | buymeacoffee.com/herbhall |

Configured via `.github/FUNDING.yml` for GitHub's native "Sponsor" button integration.

#### Supporter Recognition

Financial supporters are recognized in the product and repository:

| Tier | Threshold | Recognition |
|------|-----------|-------------|
| **Supporter** | $5+/mo or $25+ one-time | Name in `SUPPORTERS.md` + in-app About page "Community Supporters" section |
| **Backer** | $25+/mo or $100+ one-time | Above + name on project website |
| **Champion** | $100+/mo or $500+ one-time | Above + logo/link on README and website |

**In-app recognition:** The dashboard About/Settings page includes a "Community Supporters" tab displaying supporter names. This is a visible, permanent acknowledgment of community investment. Supporters list is maintained in `SUPPORTERS.md` and bundled with each release.

**Signals to acquirers:** A named list of financial supporters demonstrates genuine community investment beyond GitHub stars and download counts.

### Community Engagement & Launch Strategy

A technically excellent project with zero community engagement is invisible. The GitHub evaluation identified that all the foundational code work is meaningless if no one can find the project, understand what it does, or feel confident it actually works. These items are not optional polish -- they are adoption prerequisites.

#### Pre-Launch Checklist (Before First Public Announcement)

Every item must be complete before any public promotion (blog posts, Reddit, Hacker News):

| Item | Why It Matters |
|------|---------------|
| `v0.1.0-alpha` tagged release with binaries | "No releases" = "this doesn't work yet" in visitor perception |
| CI pipeline with passing badge in README | Proves the code compiles and tests pass -- basic credibility signal |
| Docker one-liner in README | Lowest friction path to "Time to First Value < 10 minutes" |
| At least one screenshot or GIF in README | Visitors decide in 10 seconds; a wall of text loses them |
| "Why SubNetree?" section in README | Answers the immediate question every visitor has |
| "Current Status" section in README | Honesty about what works prevents disappointment and builds trust |
| CONTRIBUTING.md | Contributors need a clear path; without it, PRs don't happen |
| 5+ labeled issues (`good first issue`, `help wanted`) | Contributors scan issues first; an empty issue tracker signals a dead project |

#### Contributor Onboarding Funnel

The goal is to convert passive visitors into active contributors. Each step reduces friction:

```
1. GitHub visitor reads README → understands value proposition (Why SubNetree?)
2. Visitor tries Docker quickstart → sees it work in < 10 minutes
3. User files a bug or feature request → issue templates guide quality reports
4. Interested dev reads CONTRIBUTING.md → knows how to set up dev environment
5. Dev picks a "good first issue" → scoped, achievable, well-described
6. Dev submits PR → PR template ensures quality, CLA bot handles IP
7. Maintainer reviews promptly → contributor feels valued, contributes again
```

**Key insight from evaluation:** An empty GitHub Issues tab and empty Discussions tab signal an inactive or abandoned project. Seeding these with genuine items (real bugs found during development, real architectural questions, real feature ideas) is not artificial -- it's making internal knowledge public.

#### Launch Announcement Strategy

Sequence matters. Announce only after the pre-launch checklist is complete.

| Channel | Timing | Content |
|---------|--------|---------|
| GitHub Release (`v0.1.0-alpha`) | Day 0 | Release notes, binary downloads, Docker image |
| Personal blog post | Day 0 | "Why I built SubNetree" -- problem statement, architecture choices, what works, what's planned |
| r/selfhosted | Day 0–1 | Show & Tell post, link to blog, Docker quickstart |
| r/homelab | Day 0–1 | Focus on HomeLab use case, hardware requirements, screenshots |
| Hacker News (Show HN) | Day 1–2 | Technical focus, architecture, BSL licensing rationale |
| Discord/Matrix | Ongoing | Real-time Q&A, feedback collection, community building |
| GitHub Discussions | Ongoing | Roadmap feedback, plugin ideas, deployment guides |

#### Community Channels

| Channel | Purpose | Phase |
|---------|---------|-------|
| GitHub Issues | Bug reports, tracked feature requests | 1 (exists) |
| GitHub Discussions | General Q&A, roadmap discussion, plugin ideas, show-your-setup | 1 |
| Discord server (or Matrix space) | Real-time chat, contributor coordination, support | 1 |
| Project website | Documentation, blog, supporter recognition | 2 |

**Discord vs. Matrix:** Discord has lower friction for most users (no account needed to browse, familiar UI). Matrix is better for open-source purists and bridge compatibility. Starting with Discord is recommended; bridge to Matrix later if demand exists.

### Acquisition Readiness Checklist

| Attribute | Requirement | Measurable Target |
|-----------|------------|-------------------|
| **Clean architecture** | Modular plugin system, clear separation of concerns, documented interfaces | Go Report Card A+ grade, architecture decision records documented |
| **Test coverage** | 70%+ across core packages, CI/CD pipeline | Codecov badge showing 70%+, green CI on main branch, < 5% flaky test rate |
| **Security posture** | Zero critical vulnerabilities, documented scan history | govulncheck + Trivy clean, Dependabot enabled, SECURITY.md published |
| **Documentation** | User guide, admin guide, plugin developer guide, API reference (OpenAPI) | 100% API endpoints documented, installation guide tested on 3+ platforms |
| **Community** | Active GitHub discussions, contributor guidelines, plugin ecosystem | 10+ contributors, < 7 day median issue response, Discord active |
| **Legal** | BSL 1.1 core, Apache 2.0 SDK, CLA, trademark, dependency audit | go-licenses clean in CI, CLA on 100% of contributions, trademark filed |
| **Adoption metrics** | Tracked and growing user base | 1,000+ stars, 1,000+ Docker pulls, 100+ verified installs (telemetry) |
| **Revenue readiness** | Clear path to monetization (not required to execute) | Pricing structure defined, license terms clear, commercial features identified |
| **Operational maturity** | Bus factor > 1, reproducible builds, release process | 10+ contributors, GoReleaser automated, SBOM on every release |

### Success Metrics & Measurement

Metrics are only useful if they are tracked consistently from early in the project lifecycle. This section defines what to measure, how to measure it, which tools to use, and what targets to aim for. Implementation is phased -- lightweight metrics start in Phase 1, deeper analytics in Phase 2+.

#### Metric Categories

##### 1. User & Community Metrics

| Metric | Source / Tool | Phase | Target (12 months) | Why It Matters |
|--------|--------------|-------|---------------------|----------------|
| GitHub Stars | GitHub Insights (built-in) | 1 | 1,000+ | Social proof and discoverability threshold |
| GitHub Forks | GitHub Insights | 1 | 100+ | Shows people building on or experimenting with the project |
| Contributors | GitHub Insights | 1 | 10+ (beyond maintainer) | Bus factor, sustainability signal |
| Issues (open/closed ratio) | GitHub Insights | 1 | Closed > open, < 7 day median response time | Responsiveness signal -- stale issues repel contributors |
| Release downloads | GitHub Releases API | 1 | 500+ cumulative across first 3 releases | Actual adoption beyond stars |
| Docker pulls | Docker Hub / GitHub Container Registry | 1 | 1,000+ | The primary installation metric for self-hosted tools |
| Discord/Matrix members | Discord server analytics | 1 | 100+ | Engaged community beyond GitHub |
| GitHub Discussions activity | GitHub Insights | 1 | 5+ threads/month with responses | Shows an active, helpful community |

##### 2. Code Quality Metrics

| Metric | Tool | Phase | Target | Why It Matters |
|--------|------|-------|--------|----------------|
| Test coverage | Codecov (free for open source) | 1 | 70%+ overall, 90%+ on core contracts | Verifiable quality signal for users and acquirers |
| Code quality grade | Go Report Card (goreportcard.com) | 1 | A+ grade | Zero-effort badge that signals Go best practices |
| Security vulnerabilities | Snyk or GitHub Dependabot + govulncheck + Trivy | 1 | Zero critical/high, < 5 medium | Clean security posture is table stakes |
| License compliance | go-licenses in CI (already planned) | 1 | Zero incompatible dependencies | Required for BSL 1.1 integrity and acquisition |
| Linter issues | golangci-lint in CI (already planned) | 1 | Zero warnings on main branch | Prevents quality erosion |
| Documentation coverage | Manual review + OpenAPI completeness | 1 | 100% of API endpoints documented | Users and integrators need complete references |
| CI pipeline health | GitHub Actions badge | 1 | Green main branch, < 5% flaky test rate | Confidence signal for contributors |

##### 3. Adoption & Growth Metrics

| Metric | Source / Tool | Phase | Target (12 months) | Why It Matters |
|--------|--------------|-------|---------------------|----------------|
| Active installations | Opt-in telemetry (anonymous ping) | 2 | 100+ verified | The only metric that proves real-world use |
| Monthly active users (MAU) | Opt-in telemetry | 2 | 50+ | Actual engagement, not just downloads |
| Retention (30-day) | Opt-in telemetry | 2 | 60%+ | Are users sticking around after first try? |
| Time to first value | User testing / dogfooding | 1 | < 10 minutes (design goal, already stated) | First experience determines adoption |
| Feature usage distribution | Opt-in telemetry | 2 | Identify top 5 features by usage | Guides development priority |
| Organic search traffic | Google Search Console (project website) | 2 | 500+ monthly impressions | Discoverability without paid promotion |
| Mentions / backlinks | GitHub search, Reddit, blog mentions | 1 | 10+ organic mentions | Word of mouth is the strongest growth signal |

##### 4. Enterprise & Acquisition Appeal Metrics

| Metric | How to Demonstrate | Phase | Why Acquirers Care |
|--------|--------------------|-------|-------------------|
| Addressable market (TAM/SAM/SOM) | Market research document | 2 | Validates revenue potential |
| Conversion path (free → paid) | Pricing page + documented funnel | 2 | Revenue predictability |
| Competitive differentiation | Feature comparison + unique capabilities | 1 (already done) | Why buy this vs. build or buy competitor |
| Switching costs | Plugin ecosystem + integrations + data lock-in | 3 | Defensibility after adoption |
| Bus factor | Contributors, documentation quality, architecture | 1 | Can it survive without the founder? |
| Technical debt | SonarQube or CodeClimate report | 2 | Integration complexity for acquirer |
| Security audit history | govulncheck + Trivy scan archive in CI | 1 | Due diligence requirement |
| CLA coverage | CLA Assistant tracking (already implemented) | 1 | Clean IP chain for acquisition |
| Architecture decision records | ADRs in `docs/adr/` | 2 | Shows deliberate, documented decisions |

#### Measurement Tools & Implementation

| Tool | Purpose | Cost | Phase | Integration Method |
|------|---------|------|-------|-------------------|
| **GitHub Insights** | Stars, forks, traffic, clones, referrers | Free (built-in) | 1 | Check Settings → Insights → Traffic weekly |
| **Codecov** | Test coverage tracking + PR comments | Free for open source | 1 | GitHub Action uploads coverage after `make test-coverage` |
| **Go Report Card** | Code quality grade + badge | Free | 1 | Register at goreportcard.com, add badge to README |
| **GitHub Dependabot** | Dependency vulnerability alerts | Free (built-in) | 1 | Enable in repository settings |
| **govulncheck** | Go-specific vulnerability scanning | Free (Go toolchain) | 1 | Already planned in CI pipeline |
| **Trivy** | Container image vulnerability scanning | Free | 1 | Already planned in CI pipeline |
| **go-licenses** | License compliance checking | Free | 1 | Already planned in CI pipeline |
| **golangci-lint** | Static analysis + linting | Free | 1 | Already planned in CI pipeline |
| **Google Search Console** | Organic search traffic for project website | Free | 2 | Register when project website launches |
| **Plausible Analytics** | Privacy-friendly website analytics | Free (self-hosted) or $9/mo | 2 | Embed in project website (NOT in the SubNetree product) |
| **Opt-in telemetry** (custom) | Active installations, MAU, feature usage | Free (custom implementation) | 2 | See Telemetry Design below |
| **SonarQube Community** | Technical debt + code smell tracking | Free (self-hosted) | 2 | Optional -- Go Report Card + golangci-lint may suffice |

#### Opt-In Telemetry Design (Phase 2)

Telemetry is the only way to measure actual adoption vs. downloads. It must be designed with privacy as the primary constraint.

**Principles:**
1. **Opt-in only.** Telemetry is disabled by default. Users explicitly enable it in settings or during the first-run wizard ("Help improve SubNetree by sharing anonymous usage data").
2. **Anonymous.** No IP addresses, hostnames, device names, credentials, or network topology sent. Ever.
3. **Transparent.** Users can see exactly what is sent before enabling. The telemetry payload is documented and viewable in the UI.
4. **Minimal.** Only aggregate data needed to answer specific questions (see payload below).
5. **No third-party services.** Telemetry data sent to a SubNetree-operated endpoint, not Google Analytics, Mixpanel, or similar.

**Telemetry payload (sent weekly):**

```json
{
  "v": 1,
  "installation_id": "random-uuid-generated-on-first-enable",
  "server_version": "1.3.2",
  "os": "linux",
  "arch": "amd64",
  "performance_profile": "medium",
  "device_count": 47,
  "agent_count": 3,
  "enabled_modules": ["recon", "pulse", "dispatch"],
  "database_driver": "sqlite",
  "uptime_hours": 168,
  "features_used": ["topology_map", "dark_mode", "scan_schedule"]
}
```

- `installation_id` is a random UUID generated when telemetry is first enabled. It is not tied to any user identity. It enables deduplication (count unique installations, not unique pings).
- No fields contain user data, device data, network data, or credentials.
- The endpoint is a simple HTTPS POST to `telemetry.subnetree.io` (or similar). The server responds with 200 OK and no body.

**Configuration:**

```yaml
telemetry:
  enabled: false                    # Opt-in only, default off
  endpoint: "https://telemetry.subnetree.io/v1/ping"
  interval: "168h"                  # Weekly
```

**Dashboard indicator:** When telemetry is enabled, a small indicator in the About page shows "Anonymous usage data sharing: enabled" with a link to the exact payload that was last sent.

#### README Badges

Badges provide at-a-glance confidence signals. Add these to the README as each becomes available:

| Badge | Source | When to Add |
|-------|--------|-------------|
| CI Build (passing/failing) | GitHub Actions | Phase 1 (when `ci.yml` is implemented) |
| Test Coverage (%) | Codecov | Phase 1 (when coverage reporting is set up) |
| Go Report Card (A+) | goreportcard.com | Phase 1 (when code quality is stable) |
| Go Version | shields.io | Phase 1 (immediately) |
| License (BSL 1.1) | shields.io | Phase 1 (immediately) |
| Latest Release | GitHub Releases | Phase 1 (when v0.1.0-alpha is tagged) |
| Docker Pulls | Docker Hub / GHCR | Phase 1 (when Docker images are published) |
| Vulnerabilities | Snyk or GitHub | Phase 2 (when security scanning is mature) |

**Badge markdown (target state):**

```markdown
[![Build](https://github.com/HerbHall/subnetree/actions/workflows/ci.yml/badge.svg)](https://github.com/HerbHall/subnetree/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/HerbHall/subnetree/branch/main/graph/badge.svg)](https://codecov.io/gh/HerbHall/subnetree)
[![Go Report Card](https://goreportcard.com/badge/github.com/HerbHall/subnetree)](https://goreportcard.com/report/github.com/HerbHall/subnetree)
[![Go Version](https://img.shields.io/github/go-mod/go-version/HerbHall/subnetree)](go.mod)
[![License](https://img.shields.io/badge/license-BSL%201.1-blue)](LICENSE)
[![Release](https://img.shields.io/github/v/release/HerbHall/subnetree)](https://github.com/HerbHall/subnetree/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/herbhall/subnetree)](https://hub.docker.com/r/herbhall/subnetree)
```

#### Milestone Targets

Concrete milestones that signal project health at each stage:

| Milestone | Target | Timeframe | Significance |
|-----------|--------|-----------|-------------|
| **v0.1.0-alpha** | First release with working discovery + dashboard | Phase 1 | "It exists and works" |
| **100 GitHub stars** | Organic growth from announcement posts | 1–3 months post-launch | Initial visibility |
| **1,000 GitHub stars** | Credibility threshold | 6–12 months | Social proof for new visitors |
| **10 contributors** | Community beyond maintainer | 6 months | Sustainability signal |
| **100 Docker pulls** | Early adoption | 1–2 months post-launch | People are trying it |
| **1,000 Docker pulls** | Real traction | 6–12 months | Consistent adoption |
| **100 active installs** | Verified via opt-in telemetry | Phase 2 | Real-world usage proof |
| **5 case studies** | User testimonials or "show your setup" posts | 12 months | Social proof for acquirers |
| **Clean security audit** | Zero critical vulnerabilities, documented scan history | Ongoing | Due diligence readiness |

#### Metrics Review Cadence

| Frequency | Review | Action |
|-----------|--------|--------|
| Weekly | GitHub traffic (Settings → Insights → Traffic), Discord activity | Note trends, respond to spikes |
| Monthly | Stars, forks, contributors, Docker pulls, issue response time, coverage % | Update internal tracking, adjust priorities |
| Quarterly | Adoption metrics (telemetry), search traffic, competitive landscape | Strategic review, roadmap adjustments |
| Per release | Download counts, bug reports, user feedback | Release retrospective, quality assessment |

#### Competitive Benchmarks

For context, here is where established players in the network monitoring space sit. SubNetree will not compete on raw numbers with mature projects, but a focused niche with strong metrics can be very attractive for acquisition.

| Project | Stars | Contributors | Docker Pulls | Key Insight |
|---------|-------|-------------|-------------|-------------|
| Prometheus | 56k+ | 900+ | 1B+ | De facto standard, massive ecosystem |
| Grafana | 65k+ | 2,000+ | 1B+ | Visualization layer, not monitoring -- complementary |
| Netdata | 70k+ | 400+ | 500M+ | Agent-first, zero-config ethos (closest to SubNetree philosophy) |
| Zabbix | 5k+ | 200+ | 100M+ | Enterprise-focused, steep learning curve |
| LibreNMS | 4k+ | 400+ | 10M+ | SNMP-focused, welcoming community, PHP stack |
| Uptime Kuma | 60k+ | 300+ | 100M+ | Beautiful UX, monitoring-only, SQLite (closest UX target) |

**SubNetree positioning:** Not competing on scale but on breadth (5-in-1: discovery + monitoring + remote access + credentials + IoT) and ease of use. A project with 1,000 stars, 10 contributors, 100 active installs, and 5 paying customers is a credible acquisition candidate if the code quality, documentation, and IP chain are clean.
