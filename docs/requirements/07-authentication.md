## Authentication and Authorization

### Phase 1: Local Authentication

- User accounts stored in SQLite with bcrypt-hashed passwords
- JWT access tokens (short-lived, 15 minutes)
- JWT refresh tokens (long-lived, 7 days, stored server-side, rotated on use)
- First-run setup wizard creates the initial admin account
- API key support for automation/scripting

### Phase 1 (Optional): OIDC/OAuth2

- Optional external identity provider support (Google, Keycloak, Authentik, Azure AD)
- Configured via YAML; disabled by default
- Auto-create local user on first OIDC login
- Map OIDC claims to NetVantage roles

### Data Model: User

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| Username | string | Login identifier |
| Email | string | Email address |
| PasswordHash | string | bcrypt hash (null for OIDC-only users) |
| Role | enum | admin, operator, viewer |
| AuthProvider | enum | local, oidc |
| OIDCSubject | string? | OIDC subject identifier |
| CreatedAt | timestamp | Account creation |
| LastLogin | timestamp | Last successful authentication |
| Disabled | bool | Account disabled flag |

### Authorization Model (Phase 1)

Three roles with fixed permissions:

| Role | Permissions |
|------|------------|
| **admin** | Full access: user management, plugin management, all CRUD |
| **operator** | Device management, scan triggers, credential use, remote sessions |
| **viewer** | Read-only access to dashboards, device list, monitoring status |

### Phase 2: RBAC

- Custom roles with granular permissions
- Per-tenant role assignments for MSP multi-tenancy
- Permission inheritance
