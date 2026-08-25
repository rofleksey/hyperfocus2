#!/usr/bin/env python3
"""Generate favicon PNGs/ICO and the OG card from the skill-check design.

Replicates web/public/favicon.svg geometry with PIL at high supersampling.
Run from repo root: python3 scripts/gen_icons.py
"""

from PIL import Image, ImageDraw, ImageFont
import math
import os

WEB_PUBLIC = os.path.join(os.path.dirname(__file__), "..", "web", "public")

# palette (must match favicon.svg)
BG = (22, 22, 28, 255)
EDGE = (61, 61, 72, 255)
RING = (85, 85, 95, 255)
GREAT = (242, 242, 244, 255)
NEEDLE = (242, 242, 244, 255)
TIP = (224, 50, 62, 255)

S = 1024  # supersample canvas


def octagon(draw, cx, cy, r):
    pts = []
    for k in range(8):
        a = math.radians(22.5 + 45 * k)
        pts.append((cx + r * math.cos(a), cy + r * math.sin(a)))
    return pts


def draw_badge(size: int) -> Image.Image:
    """Draw the skill-check badge, scaled to `size` square."""
    scale = S / 100.0
    img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    c = S / 2

    # octagonal plate
    d.polygon(octagon(d, c, c, 47 * scale), fill=BG, outline=EDGE,
              width=max(1, round(3 * scale)), )
    # PIL polygon has no joined corners; soften with a second pass
    d.polygon(octagon(d, c, c, 47 * scale), outline=EDGE)

    # base ring
    r = 26 * scale
    w = round(9 * scale)
    bbox = (c - r, c - r, c + r, c + r)
    d.arc(bbox, 0, 360, fill=RING, width=w)

    # great-zone arc: -55..-15 degrees (PIL: 0 = 3 o'clock, clockwise)
    d.arc(bbox, -55, -15, fill=GREAT, width=w)

    # needle at -35 deg: shaft to r=22, red tip to r=30.5
    a = math.radians(-35)
    x1, y1 = c, c
    x2, y2 = c + 22 * scale * math.cos(a), c + 22 * scale * math.sin(a)
    x3, y3 = c + 30.5 * scale * math.cos(a), c + 30.5 * scale * math.sin(a)
    lw = round(6 * scale)
    d.line([(x1, y1), (x2, y2)], fill=NEEDLE, width=lw)
    d.line([(x2, y2), (x3, y3)], fill=TIP, width=lw)
    rr = lw / 2
    for (px, py) in [(x1, y1), (x2, y2), (x3, y3)]:
        d.ellipse((px - rr, py - rr, px + rr, py + rr), fill=NEEDLE if py != y3 else TIP)

    # center hub
    hr = 4.5 * scale
    d.ellipse((c - hr, c - hr, c + hr, c + hr), fill=NEEDLE)

    return img.resize((size, size), Image.LANCZOS)


def font(path_candidates, size):
    for p in path_candidates:
        if os.path.exists(p):
            return ImageFont.truetype(p, size)
    raise SystemExit("no suitable font found")


def draw_og(path: str):
    W, H = 1200, 630
    img = Image.new("RGB", (W, H), (16, 16, 20))
    d = ImageDraw.Draw(img)

    # subtle radial-ish backdrop: big dim ring echoes
    cx, cy = W // 2, 240
    for rr, col in [(300, (28, 28, 34)), (210, (34, 34, 41))]:
        d.ellipse((cx - rr, cy - rr, cx + rr, cy + rr), outline=col, width=3)

    badge = draw_badge(280)
    img.paste(badge, (cx - 140, cy - 140), badge)

    dejavu = [
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
        "/usr/share/fonts/liberation-sans-fonts/LiberationSans-Bold.ttf",
    ]
    f_title = font(dejavu, 96)
    f_sub = font([
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/liberation-sans-fonts/LiberationSans-Regular.ttf",
    ], 44)
    f_url = font([
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/liberation-sans-fonts/LiberationSans-Regular.ttf",
    ], 36)

    title = "hyperfocus"
    d.text((cx, 440), title, font=f_title, fill=(242, 242, 244), anchor="mm")
    d.text((cx, 530), "Find out when you're playing against a streamer in Dead by Daylight",
           font=f_sub, fill=(170, 170, 180), anchor="mm")
    d.text((cx, 588), "hyperfocusdbd.com", font=f_url, fill=(224, 50, 62), anchor="mm")

    img.save(path, "PNG", optimize=True)


def main():
    # favicon pngs
    draw_badge(512).save(os.path.join(WEB_PUBLIC, "icon-512.png"), "PNG", optimize=True)
    draw_badge(192).save(os.path.join(WEB_PUBLIC, "icon-192.png"), "PNG", optimize=True)
    draw_badge(180).save(os.path.join(WEB_PUBLIC, "apple-touch-icon.png"), "PNG", optimize=True)
    # ico with 16/32/48
    base = draw_badge(256)
    base.save(os.path.join(WEB_PUBLIC, "favicon.ico"),
              sizes=[(16, 16), (32, 32), (48, 48)])
    draw_og(os.path.join(WEB_PUBLIC, "og-image.png"))
    print("icons written to", os.path.abspath(WEB_PUBLIC))


if __name__ == "__main__":
    main()
