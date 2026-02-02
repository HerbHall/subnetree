<required_reading>
**Read these reference files NOW:**
1. references/phase-requirements-map.md
2. references/label-taxonomy.md
3. references/issue-quality-standards.md
</required_reading>

<process>

**Step 1: Select Phase**

Ask the user which phase to generate issues for using AskUserQuestion:

<phase_options>
- **Phase 0**: Pre-development infrastructure
- **Phase 1**: Foundation (server + dashboard + discovery + topology)
- **Phase 1b**: Windows Scout Agent
- **Phase 2**: Core monitoring + multi-tenancy
- **Phase 3**: Remote access + credential vault
- **Phase 4**: Extended platform
</phase_options>

If the user already specified a phase, skip this step.

**Step 2: Read the Roadmap**

Read `docs/requirements/21-phased-roadmap.md` and extract ONLY the section for the selected phase. Identify:
- All checklist items (lines starting with `- [ ]` or `- [x]`)
- Which items are already completed (`[x]`) vs outstanding (`[ ]`)
- Any sub-sections or groupings within the phase

**Step 3: Read Relevant Requirements**

Using `references/phase-requirements-map.md`, identify which requirements doc(s) are relevant. Read ONLY the primary requirements file for the phase to gather details for issue descriptions and acceptance criteria.

**IMPORTANT**: Read ONE requirements file at a time. If the phase spans multiple docs (e.g., Phase 1 has dashboard, API, auth), read only the doc relevant to the current batch of issues.

**Step 4: Fetch Existing Issues**

Run this command to get current issues for the phase:

```bash
gh issue list --label "phase:{PHASE}" --state all --limit 200 --json number,title,state,labels
```

Parse the output to build a list of existing issue titles and their states (open/closed).

**Step 5: Diff Roadmap vs Existing Issues**

For each outstanding roadmap checklist item (`- [ ]`):
1. Search existing issues for a matching title (fuzzy match on key terms)
2. If a match exists and is open, skip it (already tracked)
3. If a match exists and is closed, check if it was completed or won't-fix'd
4. If no match exists, add it to the "issues to create" list

**Step 6: Draft Issues**

For each item in the "issues to create" list, draft an issue using the appropriate template from `templates/`:
- Use `templates/feature-issue.md` for features and enhancements
- Use `templates/epic-issue.md` for items that need multiple sub-tasks
- Use `templates/bug-issue.md` only if the item describes a known defect

For each issue, determine:
- **Title**: Action verb + specific description (see `references/issue-quality-standards.md`)
- **Body**: Fill template with description, acceptance criteria from requirements doc, and references
- **Labels**: Type + priority + module + phase (see `references/label-taxonomy.md`)
- **Milestone**: From `references/phase-requirements-map.md`

**Priority assignment heuristic:**
- Items that block other items in the phase: `P1-high`
- Core functionality for the phase's goal: `P2-medium`
- Quality improvements, docs, nice-to-haves: `P3-low`
- Security or data integrity items: `P1-high`

**Step 7: Present Batch for Review**

Present ALL drafted issues to the user in a summary table:

```
| # | Title | Labels | Priority |
|---|-------|--------|----------|
| 1 | Add device list page with sorting | feature, mod:dashboard, phase:1 | P2-medium |
| 2 | Implement WebSocket scan progress | feature, mod:core, phase:1 | P1-high |
| ...
```

Ask the user:
- "Should I create all of these issues?"
- "Do you want to modify any titles, priorities, or labels?"
- "Should any items be skipped or combined?"

Wait for approval before proceeding.

**Step 8: Create Issues**

For each approved issue, run:

```bash
gh issue create \
  --title "Issue title here" \
  --body "$(cat <<'EOF'
Issue body here...
EOF
)" \
  --label "feature,P2-medium,mod:dashboard,phase:1" \
  --milestone "Phase 1: Foundation"
```

Create issues sequentially and collect the issue numbers.

**Step 9: Report Results**

Present a summary of created issues:

```
Created N issues for Phase {PHASE}:
- #101: Add device list page with sorting
- #102: Implement WebSocket scan progress
- ...

Skipped M items (already tracked):
- #45: Implement JWT authentication (open)
- ...
```

If any epic issues were created, remind the user to update the task checklists with the sub-issue numbers.

</process>

<success_criteria>
This workflow is complete when:
- [ ] Phase selected and roadmap section read
- [ ] Relevant requirements doc read for context
- [ ] Existing issues fetched and deduplication performed
- [ ] Issue drafts reviewed and approved by user
- [ ] Issues created via `gh issue create` with proper labels and milestones
- [ ] Summary of results presented
</success_criteria>
