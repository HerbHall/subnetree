"""Generate ICO files for SubNetree Windows executables.

Draws the tree topology icon programmatically at multiple sizes
and packs them into .ico format using Pillow.

Usage: python generate-icons.py
"""

from PIL import Image, ImageDraw
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))

# Color palettes
DARK = {
    "bg": (12, 26, 14),           # #0c1a0e
    "card": (26, 46, 28),          # #1a2e1c
    "green": (74, 222, 128),       # #4ade80
    "earth": (196, 167, 125),      # #c4a77d
    "sage": (156, 163, 137),       # #9ca389
    "line_green": (74, 222, 128, 89),   # 0.35 alpha
    "line_earth": (196, 167, 125, 89),  # 0.35 alpha
    "line_faint": (74, 222, 128, 18),   # 0.07 alpha
    "ring": (74, 222, 128, 25),         # 0.10 alpha
    "sage_dim": (156, 163, 137, 127),   # 0.50 alpha
}

LIGHT = {
    "bg": (245, 245, 240),         # #f5f5f0
    "card": (245, 245, 240),       # same as bg for light
    "green": (22, 163, 74),        # #16a34a
    "earth": (146, 115, 78),       # #92734e
    "sage": (107, 122, 86),        # #6b7a56
    "line_green": (22, 101, 52, 102),   # 0.40 alpha
    "line_earth": (146, 116, 78, 102),  # 0.40 alpha
    "line_faint": (22, 101, 52, 18),    # 0.07 alpha
    "ring": (22, 101, 52, 25),          # 0.10 alpha
    "sage_dim": (100, 116, 80, 127),    # 0.50 alpha
}


def draw_icon(size, palette, rounded_bg=True):
    """Draw the SubNetree tree topology icon at the given size."""
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img, "RGBA")

    s = size / 256  # scale factor (design is 256x256)

    # Background
    if rounded_bg:
        r = int(32 * s)
        if r < 2:
            r = 2
        draw.rounded_rectangle([0, 0, size - 1, size - 1], radius=r, fill=palette["bg"])
    else:
        draw.rectangle([0, 0, size - 1, size - 1], fill=palette["bg"])

    # Helper to scale coordinates
    def sc(x, y):
        return (int(x * s), int(y * s))

    def lw(w):
        return max(1, int(w * s))

    def draw_line(x1, y1, x2, y2, color, width):
        draw.line([sc(x1, y1), sc(x2, y2)], fill=color, width=lw(width))

    def draw_node(cx, cy, outer_r, inner_r, stroke_color, fill_color, bg_color, stroke_w):
        ox, oy = sc(cx, cy)
        # Outer circle (filled bg + stroke)
        or_px = max(1, int(outer_r * s))
        ir_px = max(1, int(inner_r * s))
        sw = max(1, int(stroke_w * s))
        draw.ellipse(
            [ox - or_px, oy - or_px, ox + or_px, oy + or_px],
            fill=bg_color, outline=stroke_color, width=sw
        )
        # Inner dot
        if ir_px > 0:
            draw.ellipse(
                [ox - ir_px, oy - ir_px, ox + ir_px, oy + ir_px],
                fill=fill_color
            )

    # Skip subtle details at very small sizes
    show_details = size >= 32
    show_satellites = size >= 48
    show_ring = size >= 64

    # Outer ring
    if show_ring:
        cx, cy = sc(128, 128)
        r = int(114 * s)
        draw.ellipse([cx - r, cy - r, cx + r, cy + r], outline=palette["ring"], width=lw(2))

    # Trunk
    draw_line(128, 200, 128, 132, palette["line_green"], 5)

    # Main branches
    draw_line(128, 132, 72, 88, palette["line_green"], 4.5)
    draw_line(128, 132, 184, 88, palette["line_green"], 4.5)

    # Leaf branches
    draw_line(72, 88, 44, 56, palette["line_earth"], 3.5)
    draw_line(72, 88, 100, 56, palette["line_earth"], 3.5)
    draw_line(184, 88, 156, 56, palette["line_earth"], 3.5)
    draw_line(184, 88, 212, 56, palette["line_earth"], 3.5)

    # Discovery links
    if show_details:
        draw_line(100, 56, 156, 56, palette["line_faint"], 2)

    # Satellite lines
    if show_satellites:
        sage_line = (*palette["sage"][:3], 38)  # 0.15 alpha
        draw_line(128, 132, 96, 156, sage_line, 2)
        draw_line(128, 132, 164, 152, sage_line, 2)

    # Leaf device nodes (earth)
    for cx, cy in [(44, 56), (100, 56), (156, 56), (212, 56)]:
        draw_node(cx, cy, 10, 4.5, palette["earth"], palette["earth"], palette["card"], 3)

    # Subnet gateway nodes (green)
    for cx, cy in [(72, 88), (184, 88)]:
        draw_node(cx, cy, 13, 6, palette["green"], palette["green"], palette["card"], 4)

    # Junction node
    draw_node(128, 132, 15, 7.5, palette["green"], palette["green"], palette["card"], 4)

    # Satellite nodes (sage)
    if show_satellites:
        for cx, cy in [(96, 156), (164, 152)]:
            draw_node(cx, cy, 7, 3, palette["sage_dim"], palette["sage"], palette["card"], 2.5)

    # Root server node (glow ring + node)
    if show_ring:
        ox, oy = sc(128, 200)
        gr = int(21 * s)
        draw.ellipse([ox - gr, oy - gr, ox + gr, oy + gr], outline=palette["ring"], width=lw(2))
    draw_node(128, 200, 18, 9, palette["green"], palette["green"], palette["card"], 5)

    return img


def generate_ico(filename, palette):
    """Generate a multi-resolution ICO file."""
    sizes = [16, 24, 32, 48, 64, 128, 256]
    images = []

    for sz in sizes:
        img = draw_icon(sz, palette, rounded_bg=(sz >= 32))
        # Convert to RGBA for ICO
        images.append(img)

    # Save ICO (Pillow handles multi-size automatically)
    path = os.path.join(SCRIPT_DIR, filename)
    images[0].save(
        path,
        format="ICO",
        sizes=[(sz, sz) for sz in sizes],
        append_images=images[1:]
    )
    print(f"Created {path} ({len(sizes)} sizes: {sizes})")


def generate_pngs(prefix, palette):
    """Generate PNG files at common sizes."""
    for sz in [32, 64, 128, 256, 512]:
        img = draw_icon(sz, palette, rounded_bg=True)
        path = os.path.join(SCRIPT_DIR, f"{prefix}-{sz}.png")
        img.save(path, "PNG")
        print(f"Created {path}")


if __name__ == "__main__":
    print("Generating SubNetree icon files...\n")

    # ICO files for Windows executables
    generate_ico("subnetree.ico", DARK)
    generate_ico("subnetree-light.ico", LIGHT)
    print()

    # PNG files for various uses
    generate_pngs("icon-dark", DARK)
    generate_pngs("icon-light", LIGHT)

    print("\nDone!")
