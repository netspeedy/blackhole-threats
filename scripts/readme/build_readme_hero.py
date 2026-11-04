#!/usr/bin/env python3
"""Build the animated README hero, assets/blackhole-threats-readme-hero.svg.

The banner follows the soakes/cv README hero, like the s3ctl and zquota ones: the favicon's rings,
a two-line headline, a short body and a meta row on the left, and a generated scene on the right.
The scene is a black hole. Prefixes stream in from three kinds of feed and fall past the event
horizon while a dotted accretion disk turns around it, and blackhole announcements leave the far
side for the BGP peers, tagged with the default <local ASN>:666 community.

Edit the copy, layout or timing here and run `make readme-hero` rather than editing the
generated SVG. The headline is positioned from measured advances (see ADVANCE); if you change
its words, measure them again with measure_headline.py. Inter and JetBrains Mono (SIL Open Font
License 1.1) are fetched from Google Fonts, subset to the characters the finished SVG renders,
and embedded, so the hero renders the same on every platform. Needs network access.
"""

from __future__ import annotations

import base64
import math
import random
import re
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "assets" / "blackhole-threats-readme-hero.svg"
USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
)

# The website's red, cyan and amber (website/style.css), with the same inks as the other
# netspeedy banners so they read as one family.
BG = "#060B13"
INK = "#F1F5FA"
INK_2 = "#B5C0CD"
INK_3 = "#808D9E"
RED = "#FF4D63"
RED_2 = "#FF7A8A"
RED_3 = "#FFB3BD"
CYAN = "#5CD6FF"
CYAN_2 = "#AEEBFF"
VIOLET = "#9B73FF"
VIOLET_2 = "#CDB9FF"
AMBER = "#FFB454"
AMBER_2 = "#FFDCA8"
GREEN = "#3DD68C"

# Fonts: css class -> (Google Fonts family spec, embedded family name).
FAMILIES = {
    "sans": ("Inter:opsz,wght@14..32,400..800", "Hero Sans"),
    "mono": ("JetBrains Mono:wght@400..700", "Hero Mono"),
}
FONT_CSS = (
    ".sans{font-family:'Hero Sans',-apple-system,'Segoe UI','Helvetica Neue',Arial,sans-serif;"
    "font-optical-sizing:auto;}"
    ".mono{font-family:'Hero Mono','SF Mono',Menlo,Consolas,monospace;}"
)
TWINKLE_CSS = (
    ".tw{animation:tw 5.2s ease-in-out infinite;transform-box:fill-box;transform-origin:center}"
    ".tw1{animation-duration:6.4s}.tw2{animation-duration:4.6s}.tw3{animation-duration:7.3s}"
    "@keyframes tw{0%,100%{opacity:.85}50%{opacity:.15}}"
)

# Starfield: seeded, so rebuilding gives the same sky. No star sits behind the copy.
STAR_SEED = 11
STARS = 110
TWINKLING = 16
COPY_BOX = (36, 30, 520, 340)

# Headline advances in px, measured by measure_headline.py on Inter 800 at 44px (opsz 32,
# letter-spacing -1.3). The swash and the gradients are placed from these.
ADVANCE = {
    "Threat ": 139.49,
    "feeds": 111.43,
    "Threat feeds in.": 303.66,
    "Blackhole routes out.": 411.05,
}


# Fonts


def fetch(url: str) -> bytes:
    request = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    with urllib.request.urlopen(request, timeout=30) as response:
        return response.read()


def collect_chars(svg: str) -> dict[str, set[str]]:
    """Characters rendered by each font class, inherited through groups."""
    chars: dict[str, set[str]] = {name: set() for name in FAMILIES}

    def family_of(element: ET.Element, inherited: str | None) -> str | None:
        for cls in (element.get("class") or "").split():
            if cls in FAMILIES:
                return cls
        return inherited

    def walk(element: ET.Element, inherited: str | None, in_text: bool) -> None:
        tag = element.tag.split("}")[-1]
        family = family_of(element, inherited)
        if tag in ("style", "title", "desc", "script"):
            return
        is_text = in_text or tag in ("text", "tspan", "textPath")
        if is_text and tag in ("text", "tspan", "textPath") and element.text:
            if family is None:
                raise SystemExit(f"text without a font class: {element.text!r}")
            chars[family].update(element.text)
        for child in element:
            walk(child, family, is_text)
            if is_text and child.tail and child.tail.strip():
                chars[family or "sans"].update(child.tail)

    walk(ET.fromstring(svg), None, False)
    return chars


def font_faces(svg: str) -> str:
    """One @font-face per family, subset to exactly the characters the SVG renders."""
    chars = collect_chars(svg)
    faces = []
    for cls, (spec, local) in FAMILIES.items():
        used = chars[cls]
        if not used:
            continue
        text = "".join(sorted((used - set("\n\t\r")) | {" "}))
        query = urllib.parse.urlencode({"family": spec, "text": text, "display": "block"})
        css = fetch(f"https://fonts.googleapis.com/css2?{query}").decode()
        seen = set()
        # Google serves one variable file covering every weight; embed each file once.
        for weight, url in re.findall(
            r"font-weight:\s*([\d ]+);.*?src:\s*url\(([^)]+)\)", css, re.S
        ):
            if url in seen:
                continue
            seen.add(url)
            data = base64.b64encode(fetch(url)).decode()
            faces.append(
                f"@font-face{{font-family:'{local}';font-weight:{weight.strip()};"
                f"font-style:normal;src:url(data:font/woff2;base64,{data}) format('woff2');}}"
            )
    return "".join(faces)


# Small helpers


def esc(text: str) -> str:
    return text.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def num(value: float) -> str:
    """One decimal place, without a trailing .0."""
    text = f"{value:.1f}"
    return text[:-2] if text.endswith(".0") else text


def mono_width(chars: int, size: float, spacing: float = 0.0) -> float:
    """JetBrains Mono advances exactly 0.6 em; letter-spacing applies per glyph."""
    return chars * (size * 0.6 + spacing)


def gradient(kind: str, attrs: str, stops: list[tuple[str, str, str]]) -> list[str]:
    lines = [f"    <{kind} {attrs}>"]
    for offset, colour, opacity in stops:
        alpha = f' stop-opacity="{opacity}"' if opacity else ""
        lines.append(f'      <stop offset="{offset}" stop-color="{colour}"{alpha} />')
    return [*lines, f"    </{kind}>"]


def mix(a: str, b: str, t: float) -> str:
    """Blend two #RRGGBB colours; t=0 gives a, t=1 gives b."""
    ca = [int(a[i : i + 2], 16) for i in (1, 3, 5)]
    cb = [int(b[i : i + 2], 16) for i in (1, 3, 5)]
    return "#" + "".join(f"{round(x + (y - x) * t):02X}" for x, y in zip(ca, cb))


def chip_width(label: str, size: float = 10.5, spacing: float = 1.2) -> float:
    return mono_width(len(label), size, spacing) - spacing + 16


def chip(x: float, y: float, label: str, colour: str, cls: str = "", style: str = "") -> str:
    """A mono label in a small outlined box; (x, y) is the box's top-left corner."""
    width = chip_width(label)
    attr = f' class="{cls}"' if cls else ""
    attr += f' style="{style}"' if style else ""
    return (
        f'<g{attr}><rect x="{num(x)}" y="{num(y)}" width="{num(width)}" height="20" rx="4" '
        f'fill="#0A0F18" fill-opacity="0.9" stroke="{colour}" stroke-opacity="0.45" />'
        f'<text class="mono" x="{num(x + width / 2)}" y="{num(y + 13.8)}" text-anchor="middle" '
        f'font-size="10.5" letter-spacing="1.2" fill="{colour}">{esc(label)}</text></g>'
    )


def cubic_points(p0, p1, p2, p3, steps=80):
    pts = []
    for i in range(steps + 1):
        t = i / steps
        mt = 1 - t
        x = mt**3 * p0[0] + 3 * mt**2 * t * p1[0] + 3 * mt * t**2 * p2[0] + t**3 * p3[0]
        y = mt**3 * p0[1] + 3 * mt**2 * t * p1[1] + 3 * mt * t**2 * p2[1] + t**3 * p3[1]
        pts.append((x, y))
    return pts


def path_length(points: list[tuple[float, float]]) -> float:
    return sum(math.dist(a, b) for a, b in zip(points, points[1:]))


# Shared card


def starfield(free: list[tuple[float, float, float, float]]) -> tuple[list[str], list[str]]:
    """Background gradient and nebula defs, plus the body: base, glows and seeded stars."""
    rng = random.Random(STAR_SEED)
    stars = []
    tries = 0
    while len(stars) < STARS and tries < STARS * 50:
        tries += 1
        x, y = rng.uniform(8, 1192), rng.uniform(8, 392)
        if any(x0 <= x <= x1 and y0 <= y <= y1 for x0, y0, x1, y1 in free):
            continue
        radius = rng.choice([0.55, 0.6, 0.7, 0.8, 0.9, 1.0, 1.15])
        stars.append((x, y, radius, rng.uniform(0.18, 0.6)))
    twinkling = [
        f'<circle class="tw tw{i % 4}" cx="{num(x)}" cy="{num(y)}" r="{num(r + 0.25)}" '
        f'fill="#DDF2FF" opacity="{o + 0.25:.2f}" '
        f'style="animation-delay:-{rng.uniform(0, 6):.1f}s" />'
        for i, (x, y, r, o) in enumerate(stars[:TWINKLING])
    ]
    plain = "".join(
        f'<circle cx="{num(x)}" cy="{num(y)}" r="{num(r)}" opacity="{o:.2f}" />'
        for x, y, r, o in stars[TWINKLING:]
    )
    defs = [
        '    <linearGradient id="spaceBg" x1="0" y1="0" x2="0.35" y2="1">',
        '      <stop offset="0" stop-color="#081020" />',
        f'      <stop offset="0.55" stop-color="{BG}" />',
        '      <stop offset="1" stop-color="#0B1424" />',
        "    </linearGradient>",
        *gradient(
            "radialGradient",
            'id="nebRed" cx="0.5" cy="0.5" r="0.5"',
            [("0", RED, "0.22"), ("1", RED, "0")],
        ),
        *gradient(
            "radialGradient",
            'id="nebIndigo" cx="0.5" cy="0.5" r="0.5"',
            [("0", "#3B4BB8", "0.3"), ("1", "#3B4BB8", "0")],
        ),
    ]
    body = [
        '    <rect width="1200" height="400" fill="url(#spaceBg)" />',
        '    <ellipse cx="190" cy="-30" rx="520" ry="260" fill="url(#nebIndigo)" />',
        '    <ellipse cx="960" cy="430" rx="560" ry="300" fill="url(#nebRed)" opacity="0.5" />',
        f'    <g fill="#CFE6FF">{plain}</g>',
        f'    <g>{"".join(twinkling)}</g>',
    ]
    return defs, body


def mark_symbols() -> list[str]:
    """The rings and core from website/favicon.svg, without its tile."""
    return [
        *gradient(
            "radialGradient",
            'id="markCore" cx="0.5" cy="0.5" r="0.5"',
            [("0", RED_2, ""), ("1", RED, "")],
        ),
        '    <symbol id="mark" viewBox="0 0 64 64">',
        f'      <circle cx="32" cy="32" r="23" fill="none" stroke="{RED}" stroke-opacity="0.3" '
        'stroke-width="3" />',
        f'      <circle cx="32" cy="32" r="16" fill="none" stroke="{RED}" stroke-opacity="0.55" '
        'stroke-width="3" />',
        '      <circle cx="32" cy="32" r="9.5" fill="url(#markCore)" />',
        "    </symbol>",
    ]


def lockup() -> list[str]:
    return [
        '  <rect x="48.5" y="38.5" width="33" height="33" rx="8" fill="#0C0E14" stroke="#3B4A5F" '
        'stroke-width="1.2" />',
        '  <use href="#mark" x="49" y="39" width="32" height="32" />',
        f'  <text class="sans" x="94" y="52.5" font-size="15" font-weight="600" fill="{INK}">'
        "blackhole-threats</text>",
        f'  <text class="sans" x="94" y="69.5" font-size="12" fill="{INK_3}">'
        "netspeedy · RTBH route automation</text>",
    ]


def headline(prefix: str, word: str, suffix: str, second: str) -> tuple[list[str], list[str]]:
    """Two lines at 44px: line one with a highlighted word and an amber swash under it."""
    x = 46.0
    first = prefix + word + suffix
    start = x + ADVANCE[prefix]
    end = start + ADVANCE[word.rstrip(",.")]
    defs = [
        *gradient(
            "linearGradient",
            f'id="titleWhite" gradientUnits="userSpaceOnUse" x1="{num(x)}" y1="0" '
            f'x2="{num(x + ADVANCE[first])}" y2="0"',
            [("0", "#FFFFFF", ""), ("1", "#F2DDE2", "")],
        ),
        *gradient(
            "linearGradient",
            f'id="hl" gradientUnits="userSpaceOnUse" x1="{num(start)}" y1="0" '
            f'x2="{num(end)}" y2="0"',
            [("0", CYAN_2, ""), ("0.5", "#F4F7FA", ""), ("1", AMBER, "")],
        ),
        *gradient(
            "linearGradient",
            f'id="titleRed" gradientUnits="userSpaceOnUse" x1="{num(x)}" y1="0" '
            f'x2="{num(x + ADVANCE[second])}" y2="0"',
            [("0", RED_3, ""), ("0.5", RED_2, ""), ("1", RED, "")],
        ),
        *gradient(
            "linearGradient",
            'id="uline" x1="0" y1="0" x2="1" y2="0"',
            [("0", AMBER, "0.35"), ("0.5", AMBER, ""), ("1", AMBER, "0.5")],
        ),
    ]
    span = end - start
    els = [
        f'  <text class="sans" x="{num(x)}" y="160" font-size="44" font-weight="800" '
        'letter-spacing="-1.3" xml:space="preserve">'
        f'<tspan fill="url(#titleWhite)">{esc(prefix)}</tspan>'
        f'<tspan fill="url(#hl)">{esc(word)}</tspan>'
        f'<tspan fill="url(#titleWhite)">{esc(suffix)}</tspan></text>',
        f'  <path d="M{num(start + 2)} 172.5q{num(span * 0.5)} -6.5 {num(span - 6)} -1.5" '
        'fill="none" stroke="url(#uline)" stroke-width="2.6" stroke-linecap="round" />',
        f'  <text class="sans" x="{num(x)}" y="212" font-size="44" font-weight="800" '
        f'letter-spacing="-1.3" fill="url(#titleRed)">{esc(second)}</text>',
    ]
    return defs, els


def body(lines: list[list[tuple[str, bool]]]) -> list[str]:
    out = [f'  <text class="sans" font-size="15.5" fill="{INK_2}" xml:space="preserve">']
    for i, segments in enumerate(lines):
        spans = "".join(
            (
                f'<tspan class="mono" font-size="14" fill="{CYAN_2}">{esc(text)}</tspan>'
                if mono
                else esc(text)
            )
            for text, mono in segments
        )
        out.append(f'    <tspan x="48" y="{254 + 23 * i}">{spans}</tspan>')
    out.append("  </text>")
    return out


def meta_row(labels: list[str]) -> str:
    x = 48.0
    parts = []
    for i, label in enumerate(labels):
        if i:
            parts.append(f'<circle cx="{num(x + 9)}" cy="318.3" r="2.1" fill="{AMBER}" />')
            x += 19
        parts.append(f'<text x="{num(x)}" y="322">{esc(label)}</text>')
        x += mono_width(len(label), 11.5, 1.5)
    return (
        f'  <g class="mono" font-size="11.5" letter-spacing="1.5" fill="{INK_3}">'
        f'{"".join(parts)}</g>'
    )


# Scene: black hole


HOLE_X, HOLE_Y = 912.0, 200.0
HORIZON = 43.0  # event horizon radius
DISK = (58.0, 75.0, 94.0, 115.0, 138.0, 162.0)  # accretion ring radii (x)
TILT = 0.3  # each ring's height as a share of its width: the disk seen from just above
FEEDS = (
    ("TEXT FEED", 120.0, CYAN, "192.0.2.0/24"),
    ("JSON FEED", 200.0, VIOLET_2, "2001:db8::/32"),
    ("LOCAL FILE", 280.0, AMBER, "198.51.100.0/24"),
)
PEERS = (146.0, 200.0, 254.0)
PEER_X = 1128.0


def ring_path(rx: float) -> str:
    """A full ellipse, top half first, so a dash pattern runs round it clockwise."""
    ry = rx * TILT
    x0, x1 = HOLE_X - rx, HOLE_X + rx
    return (
        f"M{num(x0)} {num(HOLE_Y)}A{num(rx)} {num(ry)} 0 1 1 {num(x1)} {num(HOLE_Y)}"
        f"A{num(rx)} {num(ry)} 0 1 1 {num(x0)} {num(HOLE_Y)}"
    )


def ellipse_perimeter(rx: float, ry: float) -> float:
    h = ((rx - ry) / (rx + ry)) ** 2
    return math.pi * (rx + ry) * (1 + 3 * h / (10 + math.sqrt(4 - 3 * h)))


def scene_blackhole() -> tuple[list[str], str, str, list[tuple[float, float, float, float]]]:
    """Feeds falling into a black hole, blackhole routes leaving it for the BGP peers."""
    css = []

    # The accretion disk: rings of dots running round at speeds that drop with distance, the
    # inner ones hot amber and the outer ones cooling to violet. Drawn once behind the hole and
    # again in front of it, clipped to the near half.
    rings = []
    for k, rx in enumerate(DISK):
        t = k / (len(DISK) - 1)
        colour = (
            mix(AMBER_2, RED, min(1, t * 1.6)) if t < 0.6 else mix(RED, VIOLET, (t - 0.6) / 0.4)
        )
        perimeter = ellipse_perimeter(rx, rx * TILT)
        spacing = perimeter / round(perimeter / 10.5)
        speed = 46 * math.sqrt(DISK[0] / rx)  # px per second
        css.append(f"@keyframes o{k}{{to{{stroke-dashoffset:{num(-spacing)}}}}}")
        rings.append(
            f'<path d="{ring_path(rx)}" fill="none" stroke="{colour}" stroke-opacity="0.14" '
            'stroke-width="1" />'
            f'<path d="{ring_path(rx)}" fill="none" stroke="{colour}" '
            f'stroke-opacity="{0.98 - 0.5 * t:.2f}" stroke-width="{2.9 - 1.1 * t:.2f}" '
            f'stroke-linecap="round" stroke-dasharray="0.01 {spacing:.3f}" '
            f'style="animation:o{k} {spacing / speed:.3f}s linear infinite" />'
        )
    disk = "".join(rings)

    # Prefixes from each feed, falling in faster as they go and vanishing past the horizon.
    streams = []
    packets = []
    labels = []
    chips = []
    for k, (label, y, colour, prefix) in enumerate(FEEDS):
        x = 588.0
        width = chip_width(label)
        chips.append(chip(x, y - 10, label, colour, "fd", f"animation-delay:{k * 0.9 - 3.6:.2f}s"))
        p0 = (x + width + 4, y)
        p3 = (HOLE_X - 6, HOLE_Y + 3)
        p1 = (p0[0] + 70, y)
        p2 = (HOLE_X - 150, HOLE_Y + (y - HOLE_Y) * 0.22)
        pts = cubic_points(p0, p1, p2, p3)
        length = path_length(pts)
        d = (
            f"M{num(p0[0])} {num(p0[1])}C{num(p1[0])} {num(p1[1])} {num(p2[0])} {num(p2[1])} "
            f"{num(p3[0])} {num(p3[1])}"
        )
        streams.append(
            f'<path d="{d}" fill="none" stroke="{colour}" stroke-opacity="0.3" '
            'stroke-width="1.1" stroke-dasharray="2 5" />'
        )
        css.append(
            f"@keyframes fall{k}{{0%{{stroke-dashoffset:10;opacity:1}}"
            f"85%{{opacity:1}}100%{{stroke-dashoffset:{num(-length)};opacity:.2}}}}"
        )
        for j in range(3):
            dur = 3.6
            delay = k * 0.9 + j * dur / 3
            rest = -length * (0.18 + 0.27 * j + 0.08 * k)
            packets.append(
                f'<path d="{d}" fill="none" stroke="{colour}" stroke-width="4" '
                f'stroke-linecap="round" stroke-dasharray="10 {num(length + 10)}" '
                f'stroke-dashoffset="{num(rest)}" '
                f'style="animation:fall{k} {dur}s cubic-bezier(.55,0,.85,.5) infinite;'
                f'animation-delay:{delay - dur:.2f}s" />'
            )
        # Label each stream just off its line, on the side away from the disk.
        mx, my = pts[17]
        nx, ny = pts[18][0] - pts[16][0], pts[18][1] - pts[16][1]
        norm = math.hypot(nx, ny)
        side = -1 if y <= HOLE_Y else 1
        ox, oy = -ny / norm * 12 * side, nx / norm * 12 * side
        if side < 0 and oy > 0 or side > 0 and oy < 0:
            ox, oy = -ox, -oy
        labels.append(
            f'<text class="mono" x="{num(mx + ox)}" y="{num(my + oy + (3 if side > 0 else 0))}" '
            'font-size="9.5" '
            f'fill="{colour}" fill-opacity="0.75">{esc(prefix)}</text>'
        )

    # The hole: a halo, the far side of the disk lensed up over the top, the horizon itself and
    # its photon ring.
    hole = (
        f'<circle cx="{num(HOLE_X)}" cy="{num(HOLE_Y)}" r="96" fill="url(#halo)" />'
        f'<path d="M{num(HOLE_X - 68)} {num(HOLE_Y)}A68 62 0 0 1 {num(HOLE_X + 68)} {num(HOLE_Y)}" '
        'fill="none" stroke="url(#lens)" stroke-width="3" stroke-linecap="round" />'
        f'<circle cx="{num(HOLE_X)}" cy="{num(HOLE_Y)}" r="{num(HORIZON + 7)}" fill="none" '
        f'stroke="{RED}" stroke-opacity="0.22" stroke-width="9" class="glow" />'
        f'<circle cx="{num(HOLE_X)}" cy="{num(HOLE_Y)}" r="{num(HORIZON)}" fill="#010205" />'
        f'<circle cx="{num(HOLE_X)}" cy="{num(HOLE_Y)}" r="{num(HORIZON + 1.5)}" fill="none" '
        'stroke="url(#photon)" stroke-width="2.2" />'
    )

    # Blackhole routes leaving for the peers.
    announces = []
    peers = []
    for k, py in enumerate(PEERS):
        p0 = (HOLE_X + HORIZON + 6, HOLE_Y + (py - HOLE_Y) * 0.15)
        p3 = (PEER_X - 12, py)
        p1 = (HOLE_X + 120, p0[1])
        p2 = (PEER_X - 70, py)
        length = path_length(cubic_points(p0, p1, p2, p3))
        d = (
            f"M{num(p0[0])} {num(p0[1])}C{num(p1[0])} {num(p1[1])} {num(p2[0])} {num(p2[1])} "
            f"{num(p3[0])} {num(p3[1])}"
        )
        css.append(
            f"@keyframes out{k}{{from{{stroke-dashoffset:8}}to{{stroke-dashoffset:"
            f"{num(-length)}}}}}"
        )
        announces.append(
            f'<path d="{d}" fill="none" stroke="{RED}" stroke-opacity="0.4" stroke-width="1.1" />'
            f'<path d="{d}" fill="none" stroke="{RED_2}" stroke-width="3" stroke-linecap="round" '
            f'stroke-dasharray="8 {num(length + 8)}" stroke-dashoffset="{num(-length * 0.55)}" '
            f'style="animation:out{k} 2.4s linear infinite;animation-delay:{k * 0.8 - 2.4:.2f}s" />'
        )
        peers.append(
            f'<rect x="{num(PEER_X - 10)}" y="{num(py - 10)}" width="20" height="20" rx="5" '
            f'fill="#120A10" stroke="{RED_2}" stroke-opacity="0.75" stroke-width="1.2" />'
            f'<path d="M{num(PEER_X - 5)} {num(py)}h10M{num(PEER_X)} {num(py - 5)}v10" '
            f'stroke="{RED_3}" stroke-width="1.3" stroke-linecap="round" />'
            f'<circle class="ping" cx="{num(PEER_X)}" cy="{num(py)}" r="10" fill="none" '
            f'stroke="{RED_2}" stroke-width="1" '
            f'style="animation-delay:{k * 0.8 - 0.2:.2f}s" />'
        )
    peer_label = (
        f'<text class="mono" x="{num(PEER_X)}" y="{num(PEERS[0] - 22)}" text-anchor="middle" '
        f'font-size="9.5" letter-spacing="1.6" fill="{RED_3}">PEERS</text>'
    )
    community = chip(PEER_X - 42, PEERS[-1] + 26, "ASN:666", RED_2)
    summarise = chip(HOLE_X - chip_width("SUMMARISED") / 2, 296, "SUMMARISED", INK_2)

    defs = [
        *gradient(
            "radialGradient",
            'id="halo" cx="0.5" cy="0.5" r="0.5"',
            [("0", RED, "0.5"), ("0.45", RED, "0.18"), ("1", VIOLET, "0")],
        ),
        *gradient(
            "linearGradient",
            f'id="lens" gradientUnits="userSpaceOnUse" x1="{num(HOLE_X - 68)}" y1="0" '
            f'x2="{num(HOLE_X + 68)}" y2="0"',
            [
                ("0", RED, "0"),
                ("0.25", AMBER_2, "0.85"),
                ("0.75", AMBER_2, "0.85"),
                ("1", RED, "0"),
            ],
        ),
        *gradient(
            "linearGradient",
            f'id="photon" gradientUnits="userSpaceOnUse" x1="0" y1="{num(HOLE_Y - HORIZON)}" '
            f'x2="0" y2="{num(HOLE_Y + HORIZON)}"',
            [("0", AMBER_2, ""), ("0.5", RED_2, ""), ("1", RED, "")],
        ),
        *gradient(
            "radialGradient",
            'id="aura" cx="0.5" cy="0.5" r="0.5"',
            [("0", VIOLET, "0.2"), ("0.6", RED, "0.06"), ("1", RED, "0")],
        ),
        f'    <clipPath id="front"><rect x="600" y="{num(HOLE_Y)}" width="600" height="200" />'
        "</clipPath>",
    ]
    css.append(
        ".fd{animation:fd 3.6s ease-out infinite}"
        "@keyframes fd{0%{opacity:1}20%{opacity:1}35%{opacity:.6}100%{opacity:.6}}"
        ".glow{animation:glow 3s ease-in-out infinite}"
        "@keyframes glow{0%,100%{stroke-opacity:.16}50%{stroke-opacity:.34}}"
        ".ping{transform-box:fill-box;transform-origin:center;opacity:0;"
        "animation:ping 2.4s ease-out infinite}"
        "@keyframes ping{0%{opacity:.8;transform:scale(1)}30%{opacity:0;transform:scale(2.2)}"
        "100%{opacity:0;transform:scale(2.2)}}"
    )
    scene = "".join(
        [
            f'<ellipse cx="{num(HOLE_X)}" cy="{num(HOLE_Y)}" rx="300" ry="190" '
            'fill="url(#aura)" />',
            disk,
            "".join(streams),
            "".join(packets),
            hole,
            f'<g clip-path="url(#front)">{disk}</g>',
            "".join(announces),
            "".join(peers),
            peer_label,
            "".join(labels),
            "".join(chips),
            community,
            summarise,
        ]
    )
    return defs, "".join(css), scene, [(570, 40, 1180, 330)]


# Copy

CARD = {
    "title": "blackhole-threats - threat feeds in, blackhole routes out",
    "desc": (
        "A black hole with a dotted accretion disk. Prefixes from a text feed, a JSON feed and a "
        "local file fall past its event horizon, and blackhole routes leave the far side for "
        "three BGP peers, tagged with the ASN:666 community."
    ),
    "headline": ("Threat ", "feeds", " in.", "Blackhole routes out."),
    "body": [
        [("Text, JSON and local feeds, summarised into BGP", False)],
        [("blackhole routes. A failed feed keeps its last good set.", False)],
    ],
    "meta": ["FETCH", "SUMMARISE", "ANNOUNCE", "WITHDRAW"],
    "scene": scene_blackhole,
}


def build_svg(spec: dict = CARD) -> str:
    scene_defs, scene_css, scene, free = spec["scene"]()
    star_defs, star_body = starfield([COPY_BOX, *free])
    head_defs, head_els = headline(*spec["headline"])
    lines = [
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 400" width="960" height="320" '
        'role="img" aria-labelledby="title desc">',
        f'  <title id="title">{esc(spec["title"])}</title>',
        f'  <desc id="desc">{esc(spec["desc"])}</desc>',
        "",
        "  <defs>",
        "    <style>",
        "      /* Inter and JetBrains Mono, SIL Open Font License 1.1, subset to the glyphs used "
        "here. */",
        "      __FONT_FACES__",
        f"      {FONT_CSS}",
        f"      {TWINKLE_CSS}",
        f"      {scene_css}",
        "      @media (prefers-reduced-motion: reduce){",
        "        *{animation:none !important}",
        "      }",
        "    </style>",
        *star_defs,
        *mark_symbols(),
        *head_defs,
        *scene_defs,
        '    <clipPath id="frame"><rect x="1" y="1" width="1198" height="398" rx="20" />'
        "</clipPath>",
        "  </defs>",
        "",
        '  <g clip-path="url(#frame)">',
        *star_body,
        f"    {scene}",
        "  </g>",
        '  <rect x="0.75" y="0.75" width="1198.5" height="398.5" rx="20" fill="none" '
        'stroke="#1A2638" stroke-width="1.5" />',
        "",
        "  <!-- Left: brand and headline -->",
        *lockup(),
        "",
        *head_els,
        *body(spec["body"]),
        meta_row(spec["meta"]),
        "</svg>",
    ]
    return "\n".join(lines) + "\n"


def main() -> None:
    svg = build_svg()
    svg = svg.replace("__FONT_FACES__", font_faces(svg))
    if "http://" in svg.replace("http://www.w3.org/2000/svg", "") or "https://" in svg:
        raise SystemExit("external reference found in the hero SVG")
    ET.fromstring(svg)  # well-formed, or this raises
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text(svg, encoding="utf-8")
    print(f"Wrote {OUTPUT.relative_to(ROOT)} ({len(svg.encode()) / 1024:.1f} KiB)")


if __name__ == "__main__":
    main()
