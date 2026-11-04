#!/usr/bin/env python3
"""Measure Inter text advances the way a browser lays them out, for build_readme_hero.py.

Shapes the text with HarfBuzz (kerning included) on Inter's variable font at the given weight,
with opsz following font-optical-sizing:auto (the px size, clamped to 14..32), and adds CSS
letter-spacing after every character. The latin subset of Inter is downloaded from Google Fonts
on first use and cached beside this script as inter-latin.ttf (gitignored).

Needs fonttools, brotli and uharfbuzz, which build_readme_hero.py itself does not:

    python3 -m venv /tmp/hero && /tmp/hero/bin/pip install fonttools brotli uharfbuzz
    /tmp/hero/bin/python scripts/readme/measure_headline.py 44 800 -1.3 "No " "console"

usage: measure_headline.py SIZE WEIGHT LETTER_SPACING "text" ["text" ...]
"""

from __future__ import annotations

import io
import re
import sys
import urllib.request
from pathlib import Path

import uharfbuzz as hb
from fontTools.ttLib import TTFont

CACHE = Path(__file__).with_name("inter-latin.ttf")
UA = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
)


def fetch(url: str) -> bytes:
    req = urllib.request.Request(url, headers={"User-Agent": UA})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return resp.read()


def font_bytes() -> bytes:
    if not CACHE.exists():
        css = fetch(
            "https://fonts.googleapis.com/css2?family=Inter:opsz,wght@14..32,400..800"
        ).decode()
        blocks = re.findall(r"/\* (\S+) \*/\s*@font-face\s*{(.*?)}", css, re.S)
        latin = next(body for name, body in blocks if name == "latin")
        url = re.search(r"url\(([^)]+)\)", latin).group(1)
        font = TTFont(io.BytesIO(fetch(url)))
        font.flavor = None
        out = io.BytesIO()
        font.save(out)
        CACHE.write_bytes(out.getvalue())
    return CACHE.read_bytes()


def advance(text: str, size: float, weight: float, spacing: float) -> float:
    face = hb.Face(font_bytes())
    font = hb.Font(face)
    font.set_variations({"wght": weight, "opsz": min(max(size, 14), 32)})
    buf = hb.Buffer()
    buf.add_str(text)
    buf.guess_segment_properties()
    hb.shape(font, buf, {"kern": True, "liga": True, "calt": True})
    units = sum(pos.x_advance for pos in buf.glyph_positions)
    return units * size / face.upem + spacing * len(text)


if __name__ == "__main__":
    size, weight, spacing = (float(v) for v in sys.argv[1:4])
    for text in sys.argv[4:]:
        print(f"{advance(text, size, weight, spacing):8.2f}  {text!r}")
