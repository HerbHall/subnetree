# ADR-0005: Project Website with Hugo and GitHub Pages

## Status
Accepted

## Date
2025-02-02

## Context
NetVantage needs a public-facing product website to communicate the project's purpose, features, roadmap, and documentation to potential users and contributors. The project is currently unfunded, so hosting costs must be zero or near-zero. The website should be maintainable by a single developer, automatable via CI/CD, and capable of growing with the project.

The domain netvantage.net has been registered via Cloudflare Registrar (~$12/yr).

Requirements:
- Product landing page (hero, features, architecture, roadmap)
- Documentation section (getting started, contributing, architecture, FAQ)
- Blog for development updates and release announcements
- Dark theme matching the NetVantage brand identity (forest greens, earth tones)
- Automated deployment on content changes
- Zero hosting cost

## Decision
Use Hugo static site generator with the Hextra theme, deployed to GitHub Pages via GitHub Actions.

- **Hugo site source** lives in a `website/` directory within the main repository
- **Theme**: Hextra (MIT licensed), installed via Hugo Modules
- **Deployment**: GitHub Actions workflow triggered on push to `main` when `website/**` files change
- **Hosting**: GitHub Pages (free for public repositories)
- **Domain**: netvantage.net with CNAME record pointing to GitHub Pages

## Consequences

### Positive
- Zero hosting cost (GitHub Pages is free for public repos)
- Content is version-controlled alongside the code
- Hugo builds are fast (Go-based, consistent with our tech stack)
- Hextra provides landing page, documentation, and blog layouts in a single theme
- No external dependencies required (no Node.js, npm, or database)
- GitHub Actions automates deployment with zero manual intervention
- Dark mode and custom CSS support matches NetVantage brand identity
- Content is Markdown files -- easy for developers to maintain
- Site can be fully automated (future: auto-generate pages from releases, milestones)

### Negative
- Hugo Modules require Go to resolve dependencies (handled by CI, not needed locally for content edits)
- Static site limitations: no server-side forms, no dynamic content without third-party services
- Custom shortcodes needed for roadmap timeline and architecture diagrams
- Hextra is a newer theme with a smaller community than Docsy or Hugo Blox

### Neutral
- The `website/` directory adds ~50 files to the repository but is isolated from application code
- GitHub Pages enforces HTTPS automatically
- The workflow only triggers on `website/**` path changes, so application code pushes do not cause rebuilds

## Alternatives Considered

### Alternative 1: WordPress.com (herbhall.net subpage)
Product page under the existing personal website. Free, no additional hosting, but limited design control on free/personal plans. WordPress.com branding on free tier. Product page lives under a personal site, which limits professional perception. Can't install custom themes/plugins without Business plan ($25/mo).

### Alternative 2: Hugo with Docsy theme
Docsy (by Google, Apache 2.0) is battle-tested for documentation (used by Kubernetes). Excellent docs features but landing page system requires more manual work. Heavier dependency chain (Hugo Extended + Go + PostCSS + npm). Better choice if documentation is the primary concern and landing page is secondary.

### Alternative 3: Hugo with Hugo Blox theme
Hugo Blox has the best landing page capabilities (20+ content blocks). But weaker documentation features, "open core" model with some premium blocks behind paywall, and higher complexity. Better choice if marketing is the primary concern and documentation is secondary.

### Alternative 4: Separate hosting (Netlify, Vercel, Cloudflare Pages)
More features (serverless functions, edge rendering, form handling) but adds another service to manage. Not needed for a static content site. GitHub Pages is sufficient and keeps everything in one platform.
