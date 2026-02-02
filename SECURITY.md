# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Previous minor | Security fixes only |
| Older | No |

## Reporting a Vulnerability

If you discover a security vulnerability in NetVantage, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please send details to the project maintainer via the email address listed
on the GitHub profile of [@HerbHall](https://github.com/HerbHall).

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment:** Within 48 hours
- **Initial assessment:** Within 1 week
- **Fix timeline:** Depends on severity, but we aim for:
  - Critical: patch release within 72 hours
  - High: patch release within 1 week
  - Medium/Low: included in next scheduled release

### Disclosure Policy

- We will coordinate disclosure with the reporter
- We will credit reporters in release notes (unless anonymity is preferred)
- We ask reporters to allow reasonable time for a fix before public disclosure

## Security Practices

NetVantage follows these security practices:

- Dependencies are monitored via Dependabot
- CI runs `go vet` and `golangci-lint` (including `gosec`) on every PR
- Credential storage uses AES-256-GCM envelope encryption (Phase 3)
- Agent-server communication uses mTLS (Phase 1b)
- All authentication tokens use secure random generation
- No secrets are committed to the repository
