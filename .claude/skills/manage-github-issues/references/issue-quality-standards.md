<issue_quality_standards>

**Title Conventions**

- Start with a verb: "Add...", "Implement...", "Fix...", "Update...", "Remove..."
- Be specific: "Add device list page with sorting and filtering" not "Dashboard work"
- Include the module or area: "Implement JWT refresh token rotation" not "Fix auth"
- Keep under 80 characters
- Do NOT use conventional commit prefixes in titles (those are for commits)

**Body Structure**

Every issue body must include these sections:

1. **Description**: 2-3 sentences explaining what and why. Reference the requirements doc.
2. **Acceptance Criteria**: Checkboxes (`- [ ]`) listing specific, testable conditions for "done"
3. **Requirements Reference**: Link to the relevant `docs/requirements/` file and section
4. **Related Issues**: Links to blocking, blocked-by, or related issues using `#NNN`
5. **Notes** (optional): Implementation hints, design constraints, or open questions

**Acceptance Criteria Rules**

- Write from the user's or system's perspective: "User can..." or "System validates..."
- Each criterion must be independently testable
- Include both positive and negative cases where appropriate
- For API endpoints: specify HTTP method, path, request/response shape
- For UI features: specify user interactions, visual states, error handling

**Issue Sizing**

- An issue should be completable in 1-3 working sessions
- If an issue has more than 8 acceptance criteria, break it into sub-issues
- Use epic issues (with task checklists) to group related sub-issues
- Each sub-issue should be independently mergeable

**Cross-Referencing**

- Link to requirements: `See [docs/requirements/13-dashboard-architecture.md](docs/requirements/13-dashboard-architecture.md)`
- Link to related issues: `Blocked by #42` or `Related to #15`
- Link to ADRs when relevant: `Per [ADR-0002](docs/adr/0002-sqlite-first-database.md)`
- Reference PRs that implement: `Implemented in #55` (added when closing)

**Anti-Patterns to Avoid**

- Vague titles: "Fix bug", "Update code", "Improve performance"
- Missing acceptance criteria: "Make it work" is not testable
- Kitchen-sink issues: combining unrelated work in one issue
- Orphan issues: no phase label, no milestone, no module label
- Duplicate issues: always search before creating

</issue_quality_standards>
