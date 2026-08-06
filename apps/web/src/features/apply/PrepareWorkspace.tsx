"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { isAuthenticated } from "@/lib/auth";
import type { Job, PrepareResult } from "@/lib/types";

function isUuid(id: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(
    id,
  );
}

function downloadText(filename: string, content: string) {
  const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function downloadPdfFromMarkdown(title: string, markdown: string, coverLetter: string) {
  const w = window.open("", "_blank");
  if (!w) return;
  const esc = (s: string) =>
    s
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  const body = esc(markdown).replace(/\n/g, "<br/>");
  const letter = esc(coverLetter).replace(/\n/g, "<br/>");
  w.document.write(`<!doctype html><html><head><title>${esc(title)}</title>
<style>
  body{font-family:Georgia,serif;max-width:720px;margin:40px auto;padding:0 24px;color:#111;line-height:1.45}
  h1{font-size:1.4rem;margin:0 0 1rem}
  h2{font-size:1.1rem;margin:1.5rem 0 0.5rem;border-bottom:1px solid #ddd;padding-bottom:4px}
  @media print{body{margin:0}}
</style></head><body>
<h1>${esc(title)}</h1>
<div>${body}</div>
${coverLetter ? `<h2>Cover letter</h2><div>${letter}</div>` : ""}
<script>window.onload=function(){window.print()}</script>
</body></html>`);
  w.document.close();
}

export function PrepareWorkspace({ job }: { job: Job }) {
  const [authed, setAuthed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState("");
  const [tone, setTone] = useState<"professional" | "concise" | "enthusiastic">(
    "professional",
  );
  const [result, setResult] = useState<PrepareResult | null>(null);

  useEffect(() => {
    setAuthed(isAuthenticated());
  }, []);

  async function runPrepare() {
    if (!isAuthenticated()) {
      setStatus("Log in and upload your resume on Profile first.");
      return;
    }
    setBusy(true);
    setStatus("Analyzing ATS gaps and tailoring your resume…");
    try {
      const prepared = await api.prepareApplication({
        job_id: isUuid(job.id) ? job.id : undefined,
        title: job.title,
        company: job.company,
        description: job.description,
        tone,
      });
      setResult(prepared);
      if (isUuid(job.id)) {
        try {
          await api.upsertApplication(job.id, "prepared");
        } catch {
          /* optional tracking */
        }
      }
      setStatus(`Ready · model ${prepared.model || "groq"} · match ${prepared.analyze.match_score.toFixed(0)}%`);
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Prepare failed — is the API running with GROQ_API_KEY?");
    } finally {
      setBusy(false);
    }
  }

  async function copyCover() {
    const text = result?.cover_letter || "";
    if (!text) return;
    await navigator.clipboard.writeText(text);
    setStatus("Cover letter copied.");
  }

  async function copyResume() {
    const text = result?.resume_markdown || "";
    if (!text) return;
    await navigator.clipboard.writeText(text);
    setStatus("Tailored resume copied.");
  }

  return (
    <div className="container section" style={{ maxWidth: 920 }}>
      <Link href={`/jobs/${job.id}`} style={{ color: "var(--muted)", fontWeight: 700, fontSize: "0.85rem" }}>
        ← Back
      </Link>
      <h1 className="page-title" style={{ marginTop: "0.75rem" }}>
        Prepare application
      </h1>
      <p className="section-sub">
        {job.title} · {job.company}. We compare your uploaded resume to this job, show ATS gaps,
        then produce a tailored resume + cover letter you can download and paste on the employer site.
      </p>

      {!authed ? (
        <div className="notice" style={{ marginTop: "1rem" }}>
          <Link href="/login" style={{ fontWeight: 700, textDecoration: "underline" }}>
            Log in
          </Link>{" "}
          and upload a CV on{" "}
          <Link href="/profile" style={{ fontWeight: 700, textDecoration: "underline" }}>
            Profile
          </Link>{" "}
          first.
        </div>
      ) : null}

      <div style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem", marginTop: "1.25rem" }}>
        {(
          [
            ["professional", "Pro"],
            ["concise", "Short"],
            ["enthusiastic", "Bold"],
          ] as const
        ).map(([value, label]) => (
          <button
            key={value}
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={() => setTone(value)}
            style={{ opacity: tone === value ? 1 : 0.65, borderColor: tone === value ? "var(--text)" : undefined }}
          >
            {label}
          </button>
        ))}
      </div>

      <div style={{ display: "flex", flexWrap: "wrap", gap: "0.6rem", marginTop: "0.85rem" }}>
        <button type="button" className="btn btn-primary" disabled={busy || !authed} onClick={runPrepare}>
          {busy ? "Working…" : "Analyze & tailor resume"}
        </button>
        <a href={job.source_url} target="_blank" rel="noreferrer" className="btn btn-ghost">
          Open job to apply
        </a>
      </div>

      {status ? <p className="status-pill">{status}</p> : null}

      {result ? (
        <div style={{ display: "grid", gap: "1.25rem", marginTop: "1.5rem" }}>
          <section className="surface" style={{ borderRadius: 14, padding: "1.15rem" }}>
            <h2 className="section-title" style={{ fontSize: "1.15rem", margin: 0 }}>
              ATS match · {result.analyze.match_score.toFixed(0)}%
            </h2>
            <p className="section-sub" style={{ marginTop: "0.5rem" }}>
              {result.analyze.summary}
            </p>
            <div style={{ display: "grid", gap: "0.85rem", marginTop: "1rem", gridTemplateColumns: "repeat(auto-fit,minmax(200px,1fr))" }}>
              <div>
                <div style={{ fontSize: "0.78rem", fontWeight: 700, color: "var(--muted)" }}>Missing skills</div>
                <p style={{ margin: "0.35rem 0 0" }}>
                  {(result.analyze.missing_skills || []).join(", ") || "—"}
                </p>
              </div>
              <div>
                <div style={{ fontSize: "0.78rem", fontWeight: 700, color: "var(--muted)" }}>Missing keywords</div>
                <p style={{ margin: "0.35rem 0 0" }}>
                  {(result.analyze.missing_keywords || []).join(", ") || "—"}
                </p>
              </div>
              <div>
                <div style={{ fontSize: "0.78rem", fontWeight: 700, color: "var(--muted)" }}>Strengths</div>
                <p style={{ margin: "0.35rem 0 0" }}>
                  {(result.analyze.strengths || []).join(", ") || "—"}
                </p>
              </div>
            </div>
            {(result.analyze.suggestions || []).length > 0 ? (
              <ul style={{ margin: "1rem 0 0", paddingLeft: "1.1rem", color: "var(--muted)" }}>
                {result.analyze.suggestions.map((s) => (
                  <li key={s}>{s}</li>
                ))}
              </ul>
            ) : null}
          </section>

          <section className="surface" style={{ borderRadius: 14, padding: "1.15rem" }}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: "0.75rem", flexWrap: "wrap" }}>
              <h2 className="section-title" style={{ fontSize: "1.15rem", margin: 0 }}>
                Tailored resume
              </h2>
              <div style={{ display: "flex", gap: "0.45rem", flexWrap: "wrap" }}>
                <button type="button" className="btn btn-ghost btn-sm" onClick={copyResume}>
                  Copy
                </button>
                <button
                  type="button"
                  className="btn btn-ghost btn-sm"
                  onClick={() =>
                    downloadText(
                      `${job.company}-${job.title}-resume.txt`.replace(/\s+/g, "-"),
                      result.resume_markdown,
                    )
                  }
                >
                  Download .txt
                </button>
                <button
                  type="button"
                  className="btn btn-primary btn-sm"
                  onClick={() =>
                    downloadPdfFromMarkdown(
                      `${job.title} · ${job.company}`,
                      result.resume_markdown,
                      result.cover_letter,
                    )
                  }
                >
                  Print / PDF
                </button>
              </div>
            </div>
            {result.headline ? (
              <p style={{ marginTop: "0.75rem", fontWeight: 700 }}>{result.headline}</p>
            ) : null}
            <pre
              style={{
                marginTop: "0.75rem",
                whiteSpace: "pre-wrap",
                fontFamily: "ui-sans-serif, system-ui, sans-serif",
                fontSize: "0.92rem",
                lineHeight: 1.5,
                color: "var(--muted)",
              }}
            >
              {result.resume_markdown}
            </pre>
          </section>

          <section className="surface" style={{ borderRadius: 14, padding: "1.15rem" }}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: "0.75rem", flexWrap: "wrap" }}>
              <h2 className="section-title" style={{ fontSize: "1.15rem", margin: 0 }}>
                Cover letter
              </h2>
              <button type="button" className="btn btn-primary btn-sm" onClick={copyCover}>
                Copy cover letter
              </button>
            </div>
            <p style={{ marginTop: "0.75rem", whiteSpace: "pre-wrap", lineHeight: 1.55 }}>
              {result.cover_letter}
            </p>
          </section>
        </div>
      ) : null}
    </div>
  );
}
