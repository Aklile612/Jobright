#!/usr/bin/env python3
"""Stamp ATS keyword additions onto a copy of the original resume PDF (page 1 footer)."""

from __future__ import annotations

import argparse
import io
import sys

from pypdf import PdfReader, PdfWriter
from reportlab.pdfgen import canvas


def build_overlay(width: float, height: float, title: str, keywords: list[str]) -> bytes:
    buf = io.BytesIO()
    c = canvas.Canvas(buf, pagesize=(width, height))
    margin = 36
    box_h = 42
    y = 18
    c.setFillColorRGB(1, 1, 1)
    c.rect(margin - 4, y - 4, width - 2 * margin + 8, box_h, fill=1, stroke=0)
    c.setStrokeColorRGB(0.75, 0.75, 0.75)
    c.setLineWidth(0.6)
    c.line(margin, y + box_h - 6, width - margin, y + box_h - 6)

    c.setFillColorRGB(0.15, 0.15, 0.15)
    c.setFont("Helvetica-Bold", 8)
    c.drawString(margin, y + 24, f"Tailored for: {title}"[:110])
    c.setFont("Helvetica", 7.5)
    kw = ", ".join(keywords)[:220]
    if kw:
        c.drawString(margin, y + 10, f"ATS keywords emphasized: {kw}")
    c.save()
    buf.seek(0)
    return buf.read()


def stamp(src: str, dst: str, title: str, keywords: list[str]) -> None:
    reader = PdfReader(src)
    if not reader.pages:
        raise SystemExit("source PDF has no pages")
    writer = PdfWriter()
    for i, page in enumerate(reader.pages):
        if i == 0:
            box = page.mediabox
            w, h = float(box.width), float(box.height)
            overlay = PdfReader(io.BytesIO(build_overlay(w, h, title, keywords)))
            page.merge_page(overlay.pages[0])
        writer.add_page(page)
    # Keep original page count (usually 1). Never append pages.
    with open(dst, "wb") as f:
        writer.write(f)


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--src", required=True)
    p.add_argument("--dst", required=True)
    p.add_argument("--title", required=True)
    p.add_argument("--keywords", default="")
    args = p.parse_args()
    kws = [k.strip() for k in args.keywords.split(",") if k.strip()]
    stamp(args.src, args.dst, args.title, kws)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(str(e), file=sys.stderr)
        sys.exit(1)
