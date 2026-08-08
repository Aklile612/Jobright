#!/usr/bin/env python3
"""Extract plain text from a PDF. Used when pdftotext (poppler) is unavailable."""
from __future__ import annotations

import sys


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: extract_pdf_text.py <file.pdf>", file=sys.stderr)
        return 2
    path = sys.argv[1]
    try:
        import pymupdf  # type: ignore
    except ImportError:
        try:
            import fitz as pymupdf  # type: ignore
        except ImportError:
            print("pymupdf not installed", file=sys.stderr)
            return 1
    try:
        doc = pymupdf.open(path)
    except Exception as exc:  # noqa: BLE001
        print(f"open failed: {exc}", file=sys.stderr)
        return 1
    parts: list[str] = []
    for page in doc:
        parts.append(page.get_text("text") or "")
    doc.close()
    sys.stdout.write("\n".join(parts).strip())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
