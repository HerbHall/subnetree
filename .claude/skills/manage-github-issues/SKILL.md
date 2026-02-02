---
name: manage-github-issues
description: Reviews project phases and requirements to generate, audit, and triage GitHub Issues. Keeps the project on track by syncing the roadmap with actionable issues. Use when creating issues for a phase, auditing existing issues, or triaging and prioritizing open work.
---

<essential_principles>

**Issue Quality Criteria**
Every issue must be:
- **Specific**: Clear title and description that a developer can act on without additional context
- **Scoped**: One deliverable per issue; break epics into sub-tasks
- **Labeled**: Type, priority, module, and phase labels applied
- **Linked**: References the relevant requirements doc section
- **Testable**: Acceptance criteria that define "done"

**Context Conservation**
- Read ONLY the specific requirements file needed for the target phase
- Use `references/phase-requirements-map.md` to find which docs to read
- Never read multiple requirements files in a single workflow run

**Duplicate Prevention**
- Always fetch existing issues with `gh issue list` before creating new ones
- Match by title keywords and labels to detect duplicates
- Present batch to user for review before creating any issues

**Label Taxonomy (Summary)**
See `references/label-taxonomy.md` for full details.

| Category | Labels |
|----------|--------|
| Type | `feature`, `bug`, `enhancement`, `refactor`, `docs`, `test`, `chore` |
| Priority | `P0-critical`, `P1-high`, `P2-medium`, `P3-low` |
| Module | `mod:core`, `mod:recon`, `mod:pulse`, `mod:dispatch`, `mod:vault`, `mod:gateway`, `mod:scout`, `mod:dashboard` |
| Phase | `phase:0`, `phase:1`, `phase:1b`, `phase:2`, `phase:3`, `phase:4` |

**GitHub CLI Usage**
All GitHub operations use the `gh` CLI. Key commands:
- `gh issue list --label LABEL --state STATE --limit N`
- `gh issue create --title TITLE --body BODY --label L1,L2 --milestone M`
- `gh issue edit NUMBER --add-label LABEL --milestone M`
- `gh issue view NUMBER`

</essential_principles>

<intake>
What would you like to do?

1. **Generate issues for a phase** - Read the roadmap, find incomplete items, create GitHub Issues
2. **Audit existing issues** - Compare open issues against the roadmap, find gaps and problems
3. **Triage and prioritize** - Review open issues, suggest priorities, update labels and milestones

**Wait for response before proceeding.**
</intake>

<routing>
| Response | Workflow |
|----------|----------|
| 1, "generate", "create", "phase" | workflows/generate-phase-issues.md |
| 2, "audit", "review", "check", "gaps" | workflows/audit-issues.md |
| 3, "triage", "prioritize", "sort" | workflows/triage-and-prioritize.md |

**After reading the workflow, follow it exactly.**
</routing>

<reference_index>
All domain knowledge in references/:

**Project Configuration**:
- label-taxonomy.md (full label set with descriptions and colors)
- phase-requirements-map.md (which requirements docs map to which phases)

**Quality Standards**:
- issue-quality-standards.md (writing effective issues, best practices)
</reference_index>

<workflows_index>
| Workflow | Purpose |
|----------|---------|
| generate-phase-issues.md | Read roadmap phase, create missing GitHub Issues |
| audit-issues.md | Compare open issues against roadmap, find gaps |
| triage-and-prioritize.md | Review untriaged issues, suggest and apply priorities |
</workflows_index>

<templates_index>
| Template | Purpose |
|----------|---------|
| feature-issue.md | Feature or enhancement issue body |
| bug-issue.md | Bug report issue body |
| epic-issue.md | Epic/parent issue with task checklist |
</templates_index>

<success_criteria>
A successful workflow run produces:
- [ ] Relevant requirements file read for context
- [ ] Existing issues fetched and checked for duplicates
- [ ] Issues generated with proper title, body, labels, and milestone
- [ ] Batch presented to user for review before creation
- [ ] Issues created (or updates applied) via `gh` CLI
- [ ] Summary of actions taken reported to user
</success_criteria>
