<label_taxonomy>

**Type Labels** (exactly one per issue)

| Label | Color | Use When |
|-------|-------|----------|
| `feature` | `#0E8A16` | New capability that doesn't exist yet |
| `bug` | `#D73A4A` | Something is broken or behaves incorrectly |
| `enhancement` | `#A2EEEF` | Improving an existing feature |
| `refactor` | `#D4C5F9` | Code restructuring with no behavior change |
| `docs` | `#0075CA` | Documentation only |
| `test` | `#BFD4F2` | Adding or improving tests |
| `chore` | `#EDEDED` | Build, CI, tooling, dependency updates |

**Priority Labels** (exactly one per issue)

| Label | Color | Meaning |
|-------|-------|---------|
| `P0-critical` | `#B60205` | Blocks current phase; fix immediately |
| `P1-high` | `#D93F0B` | Important for current phase; resolve this sprint |
| `P2-medium` | `#FBCA04` | Should complete in current phase; schedule normally |
| `P3-low` | `#0E8A16` | Nice to have; defer if time is tight |

**Module Labels** (one or more per issue)

| Label | Scope |
|-------|-------|
| `mod:core` | Server core, config, plugin registry, event bus |
| `mod:recon` | Network scanning, device discovery |
| `mod:pulse` | Uptime monitoring, alerting |
| `mod:dispatch` | Agent management, enrollment |
| `mod:vault` | Credential storage, encryption |
| `mod:gateway` | Remote access (SSH, RDP, HTTP proxy) |
| `mod:scout` | Agent binary, metrics collection |
| `mod:dashboard` | React frontend, UI components |

**Phase Labels** (exactly one per issue)

| Label | Phase |
|-------|-------|
| `phase:0` | Pre-development infrastructure |
| `phase:1` | Foundation (server + dashboard + discovery + topology) |
| `phase:1b` | Windows Scout Agent |
| `phase:2` | Core monitoring + multi-tenancy |
| `phase:3` | Remote access + credential vault |
| `phase:4` | Extended platform (IoT, marketplace, RBAC) |

**Contributor Labels** (optional)

| Label | Meaning |
|-------|---------|
| `good first issue` | Suitable for new contributors |
| `help wanted` | Actively seeking community help |
| `mentor available` | Maintainer will guide the contributor |

**Status Labels** (optional, applied as needed)

| Label | Meaning |
|-------|---------|
| `blocked` | Cannot proceed; blocker described in comments |
| `needs-design` | Requires architectural decision before implementation |
| `needs-review` | Implementation done, awaiting review |
| `wontfix` | Intentionally not fixing; close with explanation |

</label_taxonomy>

<labeling_rules>

1. Every issue MUST have exactly one type label and one phase label
2. Every issue SHOULD have a priority label (required for phase:1 and later)
3. Every issue SHOULD have at least one module label
4. Status and contributor labels are applied as circumstances change
5. When creating issues via `gh issue create`, pass labels as comma-separated:
   `--label "feature,P2-medium,mod:dashboard,phase:1"`

</labeling_rules>
