<required_reading>
**Read these reference files NOW:**
1. references/label-taxonomy.md
2. references/issue-quality-standards.md
3. references/phase-requirements-map.md
</required_reading>

<process>

**Step 1: Determine Scope**

Ask the user what to audit using AskUserQuestion:

<audit_options>
- **All open issues** - Full audit across all phases
- **Specific phase** - Audit issues for one phase only
- **Recent issues** - Audit issues created in the last 30 days
</audit_options>

**Step 2: Fetch Issues**

Based on scope, run the appropriate `gh` command:

For all open issues:
```bash
gh issue list --state open --limit 500 --json number,title,state,labels,milestone,createdAt,updatedAt,body,assignees
```

For a specific phase:
```bash
gh issue list --state open --label "phase:{N}" --limit 200 --json number,title,state,labels,milestone,createdAt,updatedAt,body,assignees
```

**Step 3: Read Current Roadmap**

Read `docs/requirements/21-phased-roadmap.md` to understand which phase is current and what items should be tracked.

**Step 4: Run Audit Checks**

Analyze each issue against these checks:

<audit_checks>

**Labeling Completeness**
- [ ] Has exactly one type label (feature, bug, enhancement, etc.)
- [ ] Has exactly one phase label (phase:0, phase:1, etc.)
- [ ] Has at least one module label (mod:core, mod:dashboard, etc.)
- [ ] Has a priority label (P0-P3) -- required for phase:1+

**Milestone Assignment**
- [ ] Has a milestone assigned
- [ ] Milestone matches the phase label

**Issue Quality**
- [ ] Title starts with an action verb
- [ ] Title is under 80 characters
- [ ] Body contains "Acceptance Criteria" section
- [ ] Body contains at least one checkbox (`- [ ]`)
- [ ] Body references a requirements doc (contains `docs/requirements/`)

**Staleness**
- [ ] Updated within the last 30 days (if open)
- [ ] Not blocked without explanation (has `blocked` label with comment)

**Duplicates**
- [ ] No other open issue has a substantially similar title
- [ ] Check for issues that could be merged

**Roadmap Coverage**
- [ ] Each outstanding roadmap item for the current phase has a matching issue
- [ ] No orphan issues exist that don't map to any roadmap item

</audit_checks>

**Step 5: Generate Audit Report**

Present findings organized by severity:

```
## Issue Audit Report

### Issues Needing Attention (N issues)

**Missing Labels:**
- #12 "Add topology view" -- missing priority label, missing module label
- #15 "Fix login redirect" -- missing phase label

**Missing Milestone:**
- #18 "Implement rate limiting" -- no milestone assigned

**Quality Issues:**
- #20 "Dashboard stuff" -- vague title, no acceptance criteria
- #25 "Auth improvements" -- no requirements reference

**Stale Issues (no update in 30+ days):**
- #8 "Add WebSocket support" -- last updated 45 days ago

**Potential Duplicates:**
- #14 "Add device filtering" and #22 "Filter devices on list page"

### Roadmap Gaps (M items not tracked)
- Phase 1: "E2E browser tests (Playwright)" -- no matching issue
- Phase 1: "OpenAPI spec generation" -- no matching issue

### Summary
- Total open issues: X
- Issues with complete labeling: Y (Z%)
- Issues with acceptance criteria: Y (Z%)
- Stale issues: N
- Potential duplicates: N pairs
- Roadmap items without issues: M
```

**Step 6: Offer Fixes**

Ask the user which fixes to apply:

<fix_options>
1. **Add missing labels** - Apply suggested labels to under-labeled issues
2. **Assign milestones** - Set milestones based on phase labels
3. **Flag stale issues** - Add comment asking for status update
4. **Close duplicates** - Close one of each duplicate pair with cross-reference
5. **Create missing issues** - Generate issues for roadmap gaps (routes to generate-phase-issues workflow)
6. **Skip fixes** - Just report, don't change anything
</fix_options>

**Step 7: Apply Fixes**

For each approved fix, use the appropriate `gh` command:

Add labels:
```bash
gh issue edit {NUMBER} --add-label "{LABELS}"
```

Set milestone:
```bash
gh issue edit {NUMBER} --milestone "{MILESTONE}"
```

Flag stale:
```bash
gh issue comment {NUMBER} --body "This issue has had no activity for 30+ days. Is it still relevant? Please update with current status or close if no longer needed."
```

Close duplicate:
```bash
gh issue close {NUMBER} --comment "Closing as duplicate of #{OTHER}. See #{OTHER} for continued tracking."
```

**Step 8: Report Results**

Summarize all changes made:
```
Applied N fixes:
- Added labels to M issues
- Assigned milestones to M issues
- Flagged N stale issues for review
- Closed N duplicate issues
```

</process>

<success_criteria>
This workflow is complete when:
- [ ] Issues fetched for the target scope
- [ ] All audit checks run against each issue
- [ ] Audit report presented to user
- [ ] User selected which fixes to apply (or skipped)
- [ ] Approved fixes applied via `gh` CLI
- [ ] Summary of changes reported
</success_criteria>
