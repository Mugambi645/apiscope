"""Builds docs/APIScope-Tutorial.pdf. Run from anywhere: python3 docs/pdf/build.py"""
import sys
from datetime import date
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))

from reportlab.lib.pagesizes import A4
from reportlab.platypus import (BaseDocTemplate, PageTemplate, Frame, Paragraph, Spacer,
                                PageBreak, NextPageTemplate)
from reportlab.platypus.tableofcontents import TableOfContents

import theme as T
from components import Heading
from content import build_story

OUT = Path(__file__).resolve().parents[1] / "APIScope-Tutorial.pdf"
VERSION = date.today().strftime("%B %Y")
W, H = A4


class Doc(BaseDocTemplate):
    def afterFlowable(self, f):
        if isinstance(f, Heading):
            key = f"h{self.seq.nextf('bm')}"
            self.canv.bookmarkPage(key)
            self.canv.addOutlineEntry(f.plain, key, level=f.level, closed=f.level == 0)
            if f.level == 0:
                self.notify("TOCEntry", (0, f.plain, self.page, key))


def cover(c, doc):
    c.saveState()
    c.setFillColor(T.ACCENT)
    c.rect(0, H - 250, W, 250, stroke=0, fill=1)
    c.setFillColor(T.ACCENT_SOFT)
    c.setFont("Helvetica", 11)
    c.drawString(T.PAGE_MARGIN, H - 80, "STEP-BY-STEP GUIDE")
    c.setFillColor(colors_white())
    c.setFont("Helvetica-Bold", 44)
    c.drawString(T.PAGE_MARGIN, H - 140, "APIScope")
    c.setFont("Helvetica", 17)
    c.drawString(T.PAGE_MARGIN, H - 172, "A verified, runnable directory of open APIs")
    c.setFont("Helvetica", 10.5)
    c.drawString(T.PAGE_MARGIN, H - 200, "Build it, run it, secure it, extend it")
    c.setFillColor(T.INK)
    c.setFont("Helvetica", 10.5)
    y = H - 300
    for line in ["Go gateway, runner and egress proxy (standard library only)",
                 "Next.js 14 web app with a multi-language playground",
                 "Health probes, SSRF-safe proxy, Docker sandbox, manifest CI"]:
        c.setFillColor(T.ACCENT)
        c.circle(T.PAGE_MARGIN + 3, y + 3.5, 2.2, stroke=0, fill=1)
        c.setFillColor(T.INK)
        c.drawString(T.PAGE_MARGIN + 14, y, line)
        y -= 20
    c.setFillColor(T.MUTED)
    c.setFont("Helvetica", 9)
    c.drawString(T.PAGE_MARGIN, 60, f"{VERSION}  |  github.com/mugambi645/apiscope  |  MIT licence")
    c.restoreState()


def colors_white():
    from reportlab.lib import colors
    return colors.white


def page(c, doc):
    c.saveState()
    c.setStrokeColor(T.LINE)
    c.line(T.PAGE_MARGIN, 40, W - T.PAGE_MARGIN, 40)
    c.setFont("Helvetica", 8)
    c.setFillColor(T.MUTED)
    c.drawString(T.PAGE_MARGIN, 28, "APIScope tutorial")
    c.drawRightString(W - T.PAGE_MARGIN, 28, f"Page {doc.page}")
    c.restoreState()


def flatten(items):
    out = []
    for i in items:
        if isinstance(i, (list, tuple)):
            out.extend(flatten(i))
        else:
            out.append(i)
    return out


def main():
    doc = Doc(str(OUT), pagesize=A4, title="APIScope: Build, Run, Secure, Extend",
              author="Pat (mugambi645)", subject="Step-by-step guide to the APIScope project",
              leftMargin=T.PAGE_MARGIN, rightMargin=T.PAGE_MARGIN, topMargin=54, bottomMargin=56)
    frame = Frame(T.PAGE_MARGIN, 56, W - 2 * T.PAGE_MARGIN, H - 54 - 56, id="body")
    cover_frame = Frame(T.PAGE_MARGIN, 80, W - 2 * T.PAGE_MARGIN, H - 460, id="cover")
    doc.addPageTemplates([PageTemplate("cover", [cover_frame], onPage=cover),
                          PageTemplate("page", [frame], onPage=page)])

    toc = TableOfContents()
    toc.levelStyles = [T.toc1]
    toc.dotsMinLevel = 0

    story = [NextPageTemplate("page"), PageBreak(),
             Paragraph("Contents", T.h1), toc, PageBreak()]
    story += flatten(build_story(VERSION))
    doc.multiBuild(story)
    print(f"wrote {OUT} ({OUT.stat().st_size // 1024} KB)")


if __name__ == "__main__":
    main()
