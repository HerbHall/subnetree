<template name="feature_issue">

Use this template for `feature` and `enhancement` type issues.
Replace all `{PLACEHOLDER}` values with actual content.

```markdown
## Description

{2-3 sentences explaining what this feature does and why it's needed.}

**Requirements Reference:** [{REQUIREMENT_FILE}](docs/requirements/{REQUIREMENT_FILE})

## Acceptance Criteria

- [ ] {Criterion 1: specific, testable condition}
- [ ] {Criterion 2: specific, testable condition}
- [ ] {Criterion 3: specific, testable condition}
- [ ] Unit tests cover new functionality
- [ ] No new linting warnings introduced

## Related Issues

- {Blocked by #NNN / Related to #NNN / Part of #NNN}

## Notes

{Optional: implementation hints, design constraints, open questions, or links to ADRs.}
```

</template>

<guidelines>

When filling the template:
- Description should reference the specific requirements section that drives this feature
- Acceptance criteria should be 3-8 items; if more, break into sub-issues
- Always include test coverage as a criterion
- Link to related issues when known (blocking, blocked-by, parent epic)
- Notes section is optional; omit if nothing useful to add

</guidelines>
