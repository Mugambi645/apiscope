"""Reusable flowables: code blocks pulled from the real source, callouts, tables, diagram."""
import re
from pathlib import Path
from xml.sax.saxutils import escape

from reportlab.graphics.shapes import Drawing, Rect, String, Line, Polygon
from reportlab.lib import colors
from reportlab.platypus import (Paragraph, Preformatted, Table, TableStyle, Spacer,
                                KeepTogether, Flowable)
import theme as T

ROOT = Path(__file__).resolve().parents[2]


class Heading(Paragraph):
    """Heading that registers itself for the table of contents and PDF outline."""
    def __init__(self, text, style, level):
        super().__init__(text, style)
        self.level = level
        self.plain = re.sub(r"<[^>]+>", "", text)


def H1(t): return Heading(t, T.h1, 0)
def H2(t): return Heading(t, T.h2, 1)
def P(t): return Paragraph(t, T.body)
def small(t): return Paragraph(t, T.small)
def H3(t): return Paragraph(t, T.h3)


def bullets(items):
    return [Paragraph(i, T.bullet, bulletText="\u2022") for i in items]


def source(relpath, start, lines=14, title=None):
    """Excerpt `lines` lines of a real file starting at the first line containing `start`.

    Pulling from the working tree keeps the guide honest: if the code changes and the
    marker no longer matches, the build fails instead of printing stale code.
    """
    text = (ROOT / relpath).read_text().splitlines()
    for i, line in enumerate(text):
        if start in line:
            chunk = text[i:i + lines]
            break
    else:
        raise SystemExit(f"marker {start!r} not found in {relpath}")
    chunk = [l.replace("\t", "   ") for l in chunk]
    if any(len(l) > 100 for l in chunk):
        raise SystemExit(f"line too long in excerpt {relpath}:{start!r}")
    label = title or relpath
    return code_block("\n".join(chunk), label)


def code_block(text, label=None):
    pre = Preformatted(text, T.code)
    rows = []
    if label:
        rows.append([Paragraph(f"<font color='#5d6661'>{escape(label)}</font>", T.small)])
    rows.append([pre])
    t = Table(rows, colWidths=[None])
    t.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, -1), T.CODE_BG),
        ("BOX", (0, 0), (-1, -1), 0.6, T.LINE),
        ("LEFTPADDING", (0, 0), (-1, -1), 8), ("RIGHTPADDING", (0, 0), (-1, -1), 8),
        ("TOPPADDING", (0, 0), (-1, -1), 4), ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
    ]))
    return KeepTogether([t, Spacer(1, 8)])


def callout(title, text, kind="note"):
    bg, fg = {"note": (T.ACCENT_SOFT, T.ACCENT), "warn": (T.WARN_BG, T.WARN),
              "danger": (T.BAD_BG, T.BAD)}[kind]
    body = Paragraph(f"<b><font color='{fg.hexval().replace('0x', '#')}'>{title}</font></b><br/>{text}", T.body)
    t = Table([[body]], colWidths=[None])
    t.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, -1), bg), ("LINEBEFORE", (0, 0), (0, -1), 3, fg),
        ("LEFTPADDING", (0, 0), (-1, -1), 10), ("RIGHTPADDING", (0, 0), (-1, -1), 10),
        ("TOPPADDING", (0, 0), (-1, -1), 6), ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ]))
    return KeepTogether([t, Spacer(1, 8)])


def table(header, rows, widths):
    data = [[Paragraph(h, T.cell_bold) for h in header]]
    for r in rows:
        data.append([Paragraph(c, T.cell) for c in r])
    t = Table(data, colWidths=widths, repeatRows=1)
    t.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, 0), T.ACCENT_SOFT),
        ("LINEBELOW", (0, 0), (-1, 0), 0.8, T.ACCENT),
        ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.white, colors.HexColor("#f8faf9")]),
        ("GRID", (0, 0), (-1, -1), 0.4, T.LINE), ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("TOPPADDING", (0, 0), (-1, -1), 4), ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ]))
    return KeepTogether([t, Spacer(1, 8)])


class Diagram(Flowable):
    """Wraps a reportlab Drawing so it can sit in the story with a caption."""
    def __init__(self, drawing):
        super().__init__()
        self.d = drawing
        self.width, self.height = drawing.width, drawing.height

    def draw(self):
        self.d.drawOn(self.canv, 0, 0)


def _box(d, x, y, w, h, title, sub="", fill=T.ACCENT_SOFT, stroke=T.ACCENT):
    d.add(Rect(x, y, w, h, rx=6, ry=6, fillColor=fill, strokeColor=stroke, strokeWidth=1))
    d.add(String(x + w / 2, y + h - 15, title, fontName="Helvetica-Bold", fontSize=9, textAnchor="middle", fillColor=T.INK))
    if sub:
        for i, ln in enumerate(sub.split("\n")):
            d.add(String(x + w / 2, y + h - 27 - i * 10, ln, fontName="Helvetica", fontSize=7.3, textAnchor="middle", fillColor=T.MUTED))


def _arrow(d, x1, y1, x2, y2, color=T.MUTED, label=None):
    d.add(Line(x1, y1, x2, y2, strokeColor=color, strokeWidth=1.1))
    import math
    a = math.atan2(y2 - y1, x2 - x1)
    s = 5
    d.add(Polygon([x2, y2, x2 - s * math.cos(a - 0.45), y2 - s * math.sin(a - 0.45),
                   x2 - s * math.cos(a + 0.45), y2 - s * math.sin(a + 0.45)],
                  fillColor=color, strokeColor=color))
    if label:
        d.add(String((x1 + x2) / 2 + 3, (y1 + y2) / 2 + 3, label, fontName="Helvetica", fontSize=6.8, fillColor=T.MUTED))


def architecture_diagram(width=486):
    d = Drawing(width, 318)
    amber = (T.WARN_BG, T.WARN)
    # row 1: request path
    _box(d, 6, 250, 100, 52, "Browser", "search, API page\nplayground")
    _box(d, 150, 250, 120, 52, "Next.js (web)", "pages + /api rewrite\nport 3000")
    _box(d, 314, 250, 166, 52, "Gateway (Go)", "catalog, search, health\nproxy, previews, rate limits")
    _arrow(d, 106, 276, 150, 276)
    _arrow(d, 270, 276, 314, 276, label="/api/*")

    # row 2: gateway services, fed by a bus line
    xs = [6, 134, 262, 390]
    w = 96
    _box(d, xs[0], 140, w, 56, "Catalog", "JSON manifests\nvalidated at boot")
    _box(d, xs[1], 140, w, 56, "Probe worker", "uptime, latency\nHTTPS, CORS")
    _box(d, xs[2], 140, w, 56, "Try-it proxy", "catalog hosts only\nSSRF-blocked")
    _box(d, xs[3], 140, w, 56, "Runner (Go)", "docker run, limits\nport 8081", *amber)
    d.add(Line(397, 250, 397, 224, strokeColor=T.MUTED, strokeWidth=1.1))
    d.add(Line(xs[0] + w / 2, 224, xs[3] + w / 2, 224, strokeColor=T.MUTED, strokeWidth=1.1))
    for i, x in enumerate(xs):
        col = T.WARN if i == 3 else T.MUTED
        _arrow(d, x + w / 2, 224, x + w / 2, 196, color=col)
    d.add(String(xs[3] + w / 2 + 4, 210, "/api/run", fontName="Helvetica", fontSize=6.8, fillColor=T.WARN))

    # row 3: where traffic leaves
    _box(d, 6, 30, 240, 56, "Public APIs", "Open-Meteo, Nager.Date, PokeAPI ...\nreached over HTTPS only")
    _box(d, 262, 30, 100, 56, "Egress proxy", "CONNECT :443 to\ncatalog hosts", *amber)
    _box(d, 390, 30, 96, 56, "Sandbox", "node:24, python:3.12\nno direct internet", *amber)
    _arrow(d, xs[1] + w / 2, 140, xs[1] + w / 2, 86)
    _arrow(d, xs[2] + 30, 140, 226, 86)
    _arrow(d, xs[3] + w / 2, 140, xs[3] + w / 2, 86, color=T.WARN)
    d.add(String(xs[3] + w / 2 + 4, 108, "docker run", fontName="Helvetica", fontSize=6.8, fillColor=T.WARN))
    _arrow(d, 390, 58, 362, 58, color=T.WARN)
    _arrow(d, 262, 58, 246, 58, color=T.WARN)
    return Diagram(d)
