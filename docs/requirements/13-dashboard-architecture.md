## Dashboard Architecture

The dashboard is the primary interface for most users. It must be approachable enough for someone with no networking background to understand "is my network healthy?" while powerful enough for an experienced administrator to customize every aspect of their monitoring experience.

### Technology

- **Framework:** React 18+ with TypeScript
- **Build Tool:** Vite
- **Components:** shadcn/ui (Tailwind-based, copy-paste components, not a npm dependency)
- **Server State:** TanStack Query (React Query) for API data, caching, and real-time invalidation
- **Client State:** Zustand for UI state (sidebar collapsed, selected filters, theme)
- **Charts:** Recharts for time-series graphs and monitoring visualizations
- **Topology:** React Flow for interactive network topology map (zoom, pan, custom nodes, auto-layout)
- **Real-time:** WebSocket connection managed by a custom hook, invalidates TanStack Query caches
- **Routing:** React Router v6+
- **Dark Mode:** First-class support from day one (Tailwind dark: variant)

### Browser Support

| Browser | Version | Support Level |
|---------|---------|---------------|
| Chrome / Edge | Last 2 major versions | Full support |
| Firefox | Last 2 major versions | Full support |
| Safari | Last 2 major versions | Full support |
| Mobile Chrome/Safari | Last 2 major versions | Responsive support (triage-focused) |
| Internet Explorer | Any | Not supported |

### Accessibility

Target: **WCAG 2.1 AA** compliance for all dashboard pages.

- Semantic HTML elements (`nav`, `main`, `article`, `table`, etc.)
- ARIA labels for interactive elements and icon-only buttons
- Full keyboard navigation (tab order, focus indicators, skip links)
- Color contrast: minimum 4.5:1 for normal text, 3:1 for large text
- Status information conveyed by more than color alone (icons + labels + color)
- Screen reader support for data tables and alert notifications
- Reduced motion support (`prefers-reduced-motion` media query)

### Error & Empty State Patterns

Defined UX patterns for non-happy-path states:

| State | Pattern | Example |
|-------|---------|---------|
| Empty (no data yet) | Illustration + explanation + CTA | "No devices discovered. Run your first scan." |
| Loading | Skeleton placeholders (not spinners) | Shimmer cards matching final layout |
| Error (API failure) | Inline error with retry button | "Failed to load devices. Retry" |
| Connection lost | Toast notification + auto-reconnect | "Connection lost. Reconnecting..." |
| Permission denied | Explanation + redirect or contact admin | "You need operator access to view credentials." |
| No results (filtered) | Clear message + clear-filters action | "No devices match your filters. Clear filters" |

### Key UX Principles

Design for the non-technical user first, then layer in power-user capabilities. A small business owner should understand their network health at a glance. A sysadmin should be able to customize everything.

1. **Wall of Green:** When everything is healthy, the dashboard is calm (forest green background, green-400 status dots). Problems (red/amber) visually pop against the positive baseline.
2. **Information Density Gradient:** High-level status at top, progressive detail as you drill down. The default view is simple; complexity is opt-in.
3. **Search as Primary Navigation:** Fast, always-visible search bar for devices, alerts, agents. Users shouldn't need to learn a menu hierarchy.
4. **Contextual Actions:** When a device is in alert, offer immediate actions: acknowledge, connect, view history. Reduce clicks to resolution.
5. **Time Range Controls:** Every graph has "1h / 6h / 24h / 7d / 30d / custom" selectors.
6. **Customizable Everything:** Dashboard layouts, widget arrangement, alert thresholds, notification preferences, theme, and sidebar organization should all be user-configurable. Defaults are opinionated; users can override anything.
7. **Progressive Disclosure:** Show simple controls by default, reveal advanced options behind "Advanced" toggles or settings pages. Never overwhelm a first-time user.

### Dashboard Pages

| Page | Route | Description |
|------|-------|-------------|
| Setup Wizard | `/setup` | First-run: create admin, configure network, first scan |
| Dashboard | `/` | Overview: device counts by status, recent alerts, scan activity |
| Devices | `/devices` | Device list with filtering, sorting, search |
| Device Detail | `/devices/:id` | Device info, metrics, topology links, credentials, remote access |
| Device Hardware | `/devices/:id` (tab) | Hardware profile: CPU, RAM, storage, GPUs, services with collection source badges |
| Topology | `/topology` | Auto-generated network topology map |
| Monitoring | `/monitoring` | Alert list, monitoring status, metric graphs |
| Agents | `/agents` | Scout agent list, enrollment, status |
| Credentials | `/credentials` | Credential management (admin/operator only) |
| Remote Sessions | `/sessions` | Active remote sessions, launch SSH/RDP |
| Settings | `/settings` | Server config, user management, plugin management |
| About | `/about` | Version info, license, Community Supporters, system diagnostics |

### First-Run Setup Wizard

Guided flow triggered when no admin account exists. This is the single most important UX moment in the product -- it determines whether a user continues or abandons. Every step should feel obvious, with no jargon and no dead ends.

1. **Welcome** -- Product overview, what you're about to set up. Friendly tone, not technical.
2. **Create Admin Account** -- Username, email, password. Clear password requirements shown inline.
3. **Network Configuration** -- Auto-detect local subnets, show them with plain-language descriptions ("Home network: 192.168.1.0/24 -- 254 possible devices"). Allow editing for power users, but defaults should just work.
4. **First Scan** -- Trigger initial network scan with live progress. Show devices appearing in real-time as they're discovered. This is the "wow" moment.
5. **Results** -- Show discovered devices with auto-classification (router, desktop, phone, IoT, etc.). Invite user to explore. Offer guided next steps ("Set up monitoring", "Add credentials for remote access").

Goal: Under 5 minutes from first launch to seeing your network. Zero configuration required for the default experience.

### Mobile Responsiveness

Optimized for the "2 AM on-call" workflow:
- Push-capable notification support
- Summary dashboard: critical / warning / ok counts
- Device search and status view
- Acknowledge alerts and schedule downtime
- NOT a full replica of desktop -- focused on triage
