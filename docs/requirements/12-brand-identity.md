## Brand Identity & Design System

### Logo

The SubNetree logo is a **tree topology** built from network nodes, visually representing subnet hierarchy (the "Sub-Net-Tree" concept):

- **Root server node** (bottom, large green) -- the vantage point, where monitoring originates
- **Junction node** (center green) -- where the tree branches into subnets
- **2 subnet gateway nodes** (green) -- branch points representing subnet boundaries
- **4 leaf device nodes** (earth tone) -- monitored endpoints at the tree's canopy
- **2 satellite nodes** (sage) -- discovered peripheral devices
- **Branch connection lines** (green trunk, earth-tone leaves) -- network topology links
- **Subtle discovery links** (dashed) -- cross-subnet connections and scan paths
- **Outer pulse ring** (dashed) -- monitoring/discovery radar sweep

Logo files: `assets/brand/logo.svg` (dark background), `assets/brand/logo-light.svg` (light background)
Favicon: `web/public/favicon.svg`

### Color Palette

Dark mode is the default. The palette uses forest greens and earth tones.

| Role | Token | Hex | Usage |
|------|-------|-----|-------|
| **Primary accent** | `green-400` | `#4ade80` | Healthy status, primary actions, links, "online" |
| **Primary dark** | `green-600` | `#16a34a` | Buttons, active states |
| **Secondary accent** | `earth-400` | `#c4a77d` | Warm highlights, degraded status, secondary elements |
| **Tertiary** | `sage-400` | `#9ca389` | Muted text, unknown status, subtle elements |
| **Background** | `forest-950` | `#0c1a0e` | Root dark background |
| **Surface** | `forest-900` | `#0f1a10` | Page background |
| **Card** | `forest-700` | `#1a2e1c` | Card/elevated surfaces |
| **Text primary** | -- | `#f5f0e8` | Warm cream white |
| **Text secondary** | `sage-400` | `#9ca389` | Subdued content |
| **Danger** | -- | `#ef4444` | Offline status, errors, destructive actions |

### Status Color Mapping

| Status | Color | Token |
|--------|-------|-------|
| Online / Healthy | Green | `status-online` (#4ade80) |
| Degraded / Warning | Amber | `status-degraded` (#c4a77d) |
| Offline / Error | Red | `status-offline` (#ef4444) |
| Unknown | Sage | `status-unknown` (#9ca389) |

### Design Token Files

- **CSS custom properties:** `web/src/styles/design-tokens.css` (includes dark + light mode)
- **Tailwind config:** `web/tailwind.config.ts` (maps palette to Tailwind classes)

### Typography

- **Sans-serif:** System font stack (-apple-system, BlinkMacSystemFont, Segoe UI, Inter)
- **Monospace:** JetBrains Mono, Fira Code, Cascadia Code (terminal output, code, IPs)
