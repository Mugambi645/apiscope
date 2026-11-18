"""Colours, fonts and paragraph styles for the APIScope tutorial."""
from reportlab.lib import colors
from reportlab.lib.styles import ParagraphStyle
from reportlab.lib.enums import TA_LEFT

INK = colors.HexColor("#1b1f1d")
MUTED = colors.HexColor("#5d6661")
ACCENT = colors.HexColor("#0f6b4f")
ACCENT_SOFT = colors.HexColor("#e4f2ec")
LINE = colors.HexColor("#d5dcd8")
CODE_BG = colors.HexColor("#f3f5f4")
WARN = colors.HexColor("#946200")
WARN_BG = colors.HexColor("#fbf3e1")
BAD = colors.HexColor("#b3261e")
BAD_BG = colors.HexColor("#fbe9e7")

PAGE_MARGIN = 54  # 0.75 inch

body = ParagraphStyle("body", fontName="Helvetica", fontSize=10, leading=14.5,
                      textColor=INK, alignment=TA_LEFT, spaceAfter=6)
small = ParagraphStyle("small", parent=body, fontSize=8.5, leading=12, textColor=MUTED)
h1 = ParagraphStyle("h1", fontName="Helvetica-Bold", fontSize=22, leading=26,
                    textColor=ACCENT, spaceBefore=0, spaceAfter=10, keepWithNext=1)
h2 = ParagraphStyle("h2", fontName="Helvetica-Bold", fontSize=13.5, leading=17,
                    textColor=INK, spaceBefore=14, spaceAfter=5, keepWithNext=1)
h3 = ParagraphStyle("h3", fontName="Helvetica-Bold", fontSize=10.5, leading=14,
                    textColor=ACCENT, spaceBefore=8, spaceAfter=3, keepWithNext=1)
bullet = ParagraphStyle("bullet", parent=body, leftIndent=14, bulletIndent=3, spaceAfter=3)
code = ParagraphStyle("code", fontName="Courier", fontSize=7.8, leading=10.2, textColor=INK)
cell = ParagraphStyle("cell", parent=body, fontSize=8.8, leading=12, spaceAfter=0)
cell_bold = ParagraphStyle("cell_bold", parent=cell, fontName="Helvetica-Bold")
caption = ParagraphStyle("caption", parent=small, spaceBefore=3, spaceAfter=10)
toc1 = ParagraphStyle("toc1", parent=body, fontName="Helvetica-Bold", fontSize=10.5, leftIndent=0, spaceAfter=4)
toc2 = ParagraphStyle("toc2", parent=body, fontSize=9.5, leftIndent=14, spaceAfter=2, textColor=MUTED)
