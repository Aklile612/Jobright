#!/usr/bin/env python3
"""
Weave ATS keywords into the existing Technical Skills lines and project
tech stacks of a 1-page resume PDF. Keeps original layout/fonts (Lato).
No footer banners.
"""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

import pymupdf

SCRIPT_DIR = Path(__file__).resolve().parent
FONT_DIR = SCRIPT_DIR / "fonts"
LATO_BOLD = FONT_DIR / "Lato-Bold.ttf"
LATO_REG = FONT_DIR / "Lato-Regular.ttf"

SKILL_PREFIXES = [
    "Languages:",
    "Frontend:",
    "Backend:",
    "Databases:",
    "API & Tools:",
    "Other:",
]

# Route keywords into the right skill row.
ROUTES: list[tuple[str, set[str]]] = [
    (
        "Languages:",
        {
            "go", "golang", "python", "rust", "java", "javascript", "typescript",
            "kotlin", "swift", "c++", "c#", "php", "ruby", "scala", "dart",
        },
    ),
    (
        "Frontend:",
        {
            "react", "vue", "vue.js", "nuxt", "nuxt.js", "angular", "svelte",
            "tailwind", "redux", "next.js", "html", "css", "flutter",
        },
    ),
    (
        "Backend:",
        {
            "node", "node.js", "nodejs", "nestjs", "nest.js", "fastapi", "django",
            "flask", "express", "spring", "gin", "graphql", "grpc", "rest",
        },
    ),
    (
        "Databases:",
        {
            "postgres", "postgresql", "mysql", "mongodb", "redis", "firebase",
            "supabase", "sqlite", "dynamodb", "cassandra", "elasticsearch",
        },
    ),
    (
        "API & Tools:",
        {
            "stripe", "openai", "docker", "kubernetes", "k8s", "terraform",
            "ansible", "jenkins", "aws", "azure", "gcp", "vercel", "git",
            "github", "gitlab", "ci/cd", "prometheus", "grafana", "datadog",
            "helm", "kafka", "rabbitmq",
        },
    ),
]

DISPLAY = {
    "go": "Go",
    "golang": "Golang",
    "javascript": "JavaScript",
    "typescript": "TypeScript",
    "python": "Python",
    "rust": "Rust",
    "java": "Java",
    "react": "React",
    "vue": "Vue.js",
    "vue.js": "Vue.js",
    "nuxt": "Nuxt.js",
    "nuxt.js": "Nuxt.js",
    "node": "Node.js",
    "node.js": "Node.js",
    "nodejs": "Node.js",
    "nestjs": "NestJS",
    "nest.js": "NestJS",
    "fastapi": "FastAPI",
    "postgres": "PostgreSQL",
    "postgresql": "PostgreSQL",
    "redis": "Redis",
    "mongodb": "MongoDB",
    "graphql": "GraphQL",
    "docker": "Docker",
    "kubernetes": "Kubernetes",
    "k8s": "Kubernetes",
    "terraform": "Terraform",
    "ci/cd": "CI/CD",
    "devops": "DevOps",
    "observability": "Observability",
    "security": "Security",
    "testing": "Testing",
    "system design": "System Design",
    "aws": "AWS",
    "azure": "Azure",
    "gcp": "GCP",
    "kafka": "Kafka",
    "prometheus": "Prometheus",
    "grafana": "Grafana",
}


def display_name(kw: str) -> str:
    k = kw.strip().lower()
    if k in DISPLAY:
        return DISPLAY[k]
    # Title-case multi-word
    if " " in k or "/" in k:
        return "/".join(p[:1].upper() + p[1:] for p in k.split("/"))
    return kw.strip()[:1].upper() + kw.strip()[1:]


def route_prefix(kw: str) -> str:
    k = kw.strip().lower()
    for prefix, words in ROUTES:
        if k in words:
            return prefix
    return "Other:"


def already_present(hay: str, kw: str) -> bool:
    h = hay.lower()
    k = kw.lower().strip()
    aliases = {k, display_name(k).lower()}
    if k == "go":
        aliases |= {"golang", "go (golang)"}
    if k in {"kubernetes", "k8s"}:
        aliases |= {"kubernetes", "k8s"}
    if k in {"ci/cd", "cicd"}:
        aliases |= {"ci/cd", "cicd", "ci cd"}
    return any(a in h for a in aliases if a)


def collect_skill_lines(page: pymupdf.Page) -> dict[str, list[dict]]:
    """Map skill prefix -> list of line dicts {text, bbox, size, bold} in reading order."""
    found: dict[str, list[dict]] = {p: [] for p in SKILL_PREFIXES}
    blocks = page.get_text("dict")["blocks"]
    for b in blocks:
        if b.get("type") != 0:
            continue
        for line in b.get("lines", []):
            spans = line.get("spans") or []
            if not spans:
                continue
            text = "".join(s["text"] for s in spans).strip()
            if not text:
                continue
            bbox = pymupdf.Rect(line["bbox"])
            size = spans[0]["size"]
            bold = "Bold" in (spans[0].get("font") or "")
            matched = None
            for p in SKILL_PREFIXES:
                if text.startswith(p):
                    matched = p
                    break
            if matched:
                found[matched].append(
                    {"text": text, "bbox": bbox, "size": size, "bold": bold, "prefix": matched}
                )
            else:
                # Continuation of Other: (no prefix) sitting under skills area
                if found["Other:"] and bbox.y0 >= found["Other:"][0]["bbox"].y0 - 2:
                    # likely wrap of Other
                    last = found["Other:"][-1]
                    if bbox.y0 > last["bbox"].y0 + 2 and not any(
                        text.startswith(p) for p in SKILL_PREFIXES + ["Education", "Experience", "Projects", "Technical"]
                    ):
                        found["Other:"].append(
                            {
                                "text": text,
                                "bbox": bbox,
                                "size": size,
                                "bold": bold,
                                "prefix": "Other:",
                                "continuation": True,
                            }
                        )
    return found


def collect_project_titles(page: pymupdf.Page) -> list[dict]:
    """Project title lines that include a tech stack after '|'."""
    out = []
    in_projects = False
    blocks = page.get_text("dict")["blocks"]
    for b in blocks:
        if b.get("type") != 0:
            continue
        for line in b.get("lines", []):
            spans = line.get("spans") or []
            text = "".join(s["text"] for s in spans).strip()
            if not text:
                continue
            if text == "Projects" or text.startswith("Projects"):
                in_projects = True
                continue
            if text.startswith("Technical Skills"):
                in_projects = False
                continue
            if not in_projects:
                continue
            if "|" in text and "Bold" in (spans[0].get("font") or ""):
                out.append(
                    {
                        "text": text,
                        "bbox": pymupdf.Rect(line["bbox"]),
                        "size": spans[0]["size"],
                        "bold": True,
                    }
                )
    return out


def merge_skills(skill_lines: dict[str, list[dict]], keywords: list[str]) -> dict[str, str]:
    """Return new full line text per prefix (including prefix label)."""
    # Join current text per prefix
    current: dict[str, str] = {}
    for p in SKILL_PREFIXES:
        parts = [x["text"] for x in skill_lines.get(p) or []]
        if not parts:
            continue
        # First line has prefix; continuations don't
        joined = parts[0]
        for cont in parts[1:]:
            if cont.startswith(p):
                joined = cont
            else:
                joined = joined.rstrip(", ") + ", " + cont
        current[p] = joined

    full_blob = " ".join(current.values()).lower()
    for kw in keywords:
        if already_present(full_blob, kw):
            continue
        prefix = route_prefix(kw)
        label = display_name(kw)
        if prefix not in current:
            # create Other if missing
            prefix = "Other:" if "Other:" in current else next(iter(current), "Other:")
        if already_present(current.get(prefix, ""), kw):
            continue
        line = current.get(prefix, prefix)
        # Insert new keyword right after the label so fit-to-rows keeps ATS terms
        # when trimming (drops trailing original fluff first).
        if ":" in line:
            head, body = line.split(":", 1)
            body = body.strip().strip(",")
            line = f"{head}: {label}" + (f", {body}" if body else "")
        else:
            line = line.rstrip() + ", " + label
        current[prefix] = line
        full_blob = " ".join(current.values()).lower()
    return current


def merge_projects(projects: list[dict], keywords: list[str], limit: int = 4) -> list[tuple[dict, str]]:
    """Add a few keywords into project tech stacks (1 per project to avoid overflow)."""
    updates = []
    remaining = []
    for kw in keywords:
        k = kw.lower()
        if k in {
            "docker", "kubernetes", "k8s", "terraform", "redis", "postgres",
            "postgresql", "ci/cd", "aws", "gcp", "azure", "kafka", "prometheus",
            "grafana", "devops",
        }:
            remaining.append(kw)
    if not remaining:
        remaining = list(keywords)[:limit]
    remaining = remaining[:limit]

    used: set[str] = set()
    for proj in projects:
        text = proj["text"]
        # Skip already-long stacks
        if len(text) > 110:
            continue
        add = None
        for kw in remaining:
            if kw.lower() in used:
                continue
            if already_present(text, kw):
                used.add(kw.lower())
                continue
            add = display_name(kw)
            used.add(kw.lower())
            break
        if not add:
            continue
        if "|" in text:
            left, right = text.split("|", 1)
            right = right.strip().rstrip(",")
            new_text = f"{left.strip()} | {right}, {add}"
        else:
            new_text = text + " | " + add
        updates.append((proj, new_text))
        remaining = [k for k in remaining if k.lower() not in used]
        if not remaining:
            break
    return updates


def text_width(text: str, fontname: str, fontsize: float, fontfile: str | None = None) -> float:
    try:
        if fontfile:
            return pymupdf.get_text_length(text, fontname=fontname, fontsize=fontsize, fontfile=fontfile)
        return pymupdf.get_text_length(text, fontname=fontname, fontsize=fontsize)
    except Exception:
        return len(text) * fontsize * 0.51


def wrap_rows(text: str, fontname: str, size: float, max_w: float, fontfile: str | None) -> list[str]:
    words = text.split()
    rows: list[str] = []
    cur = ""
    for w in words:
        trial = (cur + " " + w).strip()
        if text_width(trial, fontname, size, fontfile) <= max_w or not cur:
            cur = trial
        else:
            rows.append(cur)
            cur = w
    if cur:
        rows.append(cur)
    return rows


def fit_to_rows(
    text: str,
    fontname: str,
    size: float,
    max_w: float,
    max_rows: int,
    fontfile: str | None,
) -> list[str]:
    """Wrap text; if it exceeds max_rows, drop trailing comma-items until it fits."""
    rows = wrap_rows(text, fontname, size, max_w, fontfile)
    if len(rows) <= max_rows:
        return rows
    # Drop from the end of the skill/project list (after the last comma groups)
    parts = [p.strip() for p in text.split(",")]
    while len(parts) > 1:
        parts = parts[:-1]
        candidate = ", ".join(parts)
        rows = wrap_rows(candidate, fontname, size, max_w, fontfile)
        if len(rows) <= max_rows:
            return rows
    return rows[:max_rows]


def ensure_font(page: pymupdf.Page, bold: bool) -> tuple[str, str | None]:
    fontfile = str(LATO_BOLD if bold and LATO_BOLD.exists() else LATO_REG if LATO_REG.exists() else "")
    if not fontfile:
        return "helv", None
    fontname = "latoBold" if bold else "latoReg"
    page.insert_font(fontname=fontname, fontfile=fontfile)
    return fontname, fontfile


def tailor(src: str, dst: str, title: str, keywords: list[str]) -> None:
    _ = title  # CLI compat — never printed on the PDF
    doc = pymupdf.open(src)
    if doc.page_count < 1:
        raise SystemExit("source PDF has no pages")
    page = doc[0]
    page_width = float(page.rect.width)

    skills = collect_skill_lines(page)
    projects = collect_project_titles(page)
    new_skills = merge_skills(skills, keywords)
    proj_updates = merge_projects(projects, keywords)

    writes: list[dict] = []

    for proj, new_text in proj_updates:
        r = pymupdf.Rect(proj["bbox"])
        r.x1 = page_width - 18
        r.x0 = max(0, r.x0 - 1)
        r.y0 -= 1
        r.y1 += 1
        page.add_redact_annot(r, fill=(1, 1, 1))
        writes.append(
            {
                "kind": "project",
                "x": float(proj["bbox"].x0),
                "y": float(proj["bbox"].y1 - 1.5),
                "size": float(proj["size"]),
                "bold": True,
                "text": new_text,
                "max_rows": 1,
                "line_h": float(proj["bbox"].height + 2),
            }
        )

    for prefix, new_text in new_skills.items():
        lines = skills.get(prefix) or []
        if not lines:
            continue
        old = " ".join(x["text"] for x in lines)
        if old.strip() == new_text.strip():
            continue
        for ln in lines:
            r = pymupdf.Rect(ln["bbox"])
            r.x1 = page_width - 18
            r.x0 = max(0, r.x0 - 1)
            r.y0 -= 1
            r.y1 += 1
            page.add_redact_annot(r, fill=(1, 1, 1))
        first = lines[0]
        line_h = float(first["bbox"].height + 2)
        if len(lines) >= 2:
            line_h = max(line_h, float(lines[1]["bbox"].y0 - lines[0]["bbox"].y0))
        # Stay within original vertical footprint (keeps 1 page)
        max_rows = max(1, len(lines))
        writes.append(
            {
                "kind": "skill",
                "x": float(first["bbox"].x0),
                "y": float(first["bbox"].y1 - 1.5),
                "size": float(first["size"]),
                "bold": bool(first["bold"]),
                "text": new_text,
                "max_rows": max_rows,
                "line_h": line_h,
            }
        )

    page.apply_redactions(images=0, graphics=0)

    for w in writes:
        fontname, fontfile = ensure_font(page, w["bold"])
        max_w = page_width - 36 - w["x"]
        rows = fit_to_rows(w["text"], fontname, w["size"], max_w, w["max_rows"], fontfile)
        for i, row in enumerate(rows):
            y = w["y"] + i * w["line_h"]
            if y > page.rect.height - 20:
                break
            kwargs = dict(
                fontname=fontname,
                fontsize=w["size"],
                color=(0.12, 0.12, 0.12),
                overlay=True,
            )
            if fontfile:
                kwargs["fontfile"] = fontfile
            page.insert_text((w["x"], y), row, **kwargs)

    os.makedirs(os.path.dirname(dst) or ".", exist_ok=True)
    doc.save(dst, garbage=4, deflate=True)
    doc.close()


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--src", required=True)
    p.add_argument("--dst", required=True)
    p.add_argument("--title", default="")
    p.add_argument("--keywords", default="")
    args = p.parse_args()
    kws = [k.strip() for k in args.keywords.split(",") if k.strip()]
    seen: set[str] = set()
    clean: list[str] = []
    for k in kws:
        key = k.lower()
        if key in seen:
            continue
        seen.add(key)
        clean.append(k)
    tailor(args.src, args.dst, args.title, clean)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(str(e), file=sys.stderr)
        sys.exit(1)
