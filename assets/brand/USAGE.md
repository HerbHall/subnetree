# SubNetree Logo Usage Guidelines

## Available Assets

| File | Format | Purpose |
|------|--------|---------|
| `logo.svg` | SVG | Full logo, dark background variant |
| `logo-light.svg` | SVG | Full logo, light background variant |
| `icon-square-dark.svg` | SVG | Square icon, dark background |
| `icon-square-light.svg` | SVG | Square icon, light background |
| `icon-dark-{32,64,128,256,512}.png` | PNG | Raster dark variants at standard sizes |
| `icon-light-{32,64,128,256,512}.png` | PNG | Raster light variants at standard sizes |
| `subnetree.ico` | ICO | Windows icon, dark variant |
| `subnetree-light.ico` | ICO | Windows icon, light variant |
| `social-card.svg` | SVG | OpenGraph social card (1200x630) |

## Display Size

- **Icon (square):** minimum 32x32 pixels
- **Full logo:** minimum 120x120 pixels
- **Social card:** display at 1200x630 or proportional scale

## Clear Space

Maintain a minimum clear space of **25% of the logo width** on all sides. No other graphic elements, text, or borders should encroach on this area.

```text
         ┌──────────────────────┐
         │                      │
         │    ┌────────────┐    │
         │    │            │    │
         │    │   LOGO     │    │
         │    │            │    │
         │    └────────────┘    │
         │                      │
         └──────────────────────┘
               25% margin
```

## Background Usage

- Use the **dark variant** (`logo.svg`, `icon-dark-*.png`) on dark backgrounds (`#0c1a0e`, `#0f1a10`, or similar dark tones)
- Use the **light variant** (`logo-light.svg`, `icon-light-*.png`) on light backgrounds (`#f5f0e8`, white, or similar light tones)
- Never place the dark variant on a light background or vice versa

## Brand Colors

| Color | Hex | Usage |
|-------|-----|-------|
| Primary green | `#4ade80` | Node highlights, accents |
| Primary dark green | `#16a34a` | Hover states, emphasis |
| Earth tone | `#c4a77d` | Leaf nodes, secondary accents |
| Sage | `#9ca389` | Subtle text, peripheral elements |
| Background dark | `#0c1a0e` | Dark mode background |
| Surface | `#0f1a10` | Card surfaces on dark |
| Card | `#1a2e1c` | Elevated surfaces on dark |
| Text primary | `#f5f0e8` | Primary text on dark backgrounds |

## Restrictions

- **Do not** stretch, skew, or distort the logo
- **Do not** rotate the logo
- **Do not** recolor or alter the logo colors
- **Do not** add drop shadows, outlines, or effects
- **Do not** place the logo on busy or low-contrast backgrounds
- **Do not** crop or partially obscure the logo
- **Do not** display the logo smaller than the minimum sizes listed above

## Regenerating Raster Assets

Run the icon generation script to regenerate PNG files from the SVG sources:

```bash
cd assets/brand
python generate-icons.py
```

This requires Python with the Pillow and cairosvg libraries installed.
