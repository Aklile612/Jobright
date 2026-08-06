"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, API_URL } from "@/lib/api";
import { getToken, isAuthenticated } from "@/lib/auth";
import type { AnalyzeResult, Job, PrepareResult } from "@/lib/types";

function isUuid(id: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(
    id,
  );
}

async function downloadAuthed(path: string, filename: string) {
  const token = getToken();
  const res = await fetch(`${API_URL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) {
    throw new Error("Download failed");
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function PrepareWorkspace({ job }: { job: Job }) {
  const [authed, setAuthed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState("");
  const [tone, setTone] = useState<"professional" | "concise" | "enthusiastic">(
    "professional",
  );
  const [analyze, setAnalyze] = useState<AnalyzeResult | null>(null);
  const [result, setResult] = useState<PrepareResult | null>(null);

  useEffect(() => {
    setAuthed(isAuthenticated());
  }, []);

  const jobPayload = {
    job_id: isUuid(job.id) ? job.id : undefined,
    title: job.title,
    company: job.company,
    description: job.description,
  };

  async function runAnalyze() {
    if (!isAuthenticated()) {
      setStatus("Log in and upload your resume on Profile first.");
      return;
    }
    setBusy(true);
    setStatus("Scoring keyword coverage against this job…");
    setResult(null);
    try {
      const scored = await api.analyzeJob(jobPayload);
      setAnalyze(scored);
      const cov =
        scored.covered != null && scored.total_keywords
          ? ` · ${scored.covered}/${scored.total_keywords} keywords`
          : "";
      setStatus(`ATS match ${scored.match_score.toFixed(0)}%${cov} — then tailor your PDF`);
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Analyze failed");
    } finally {
      setBusy(false);
    }
  }

  async function runTailor() {
    if (!isAuthenticated()) {
      setStatus("Log in and upload your resume on Profile first.");
      return;
    }
    if (!analyze) {
      setStatus("Run Analyze first to see missing keywords.");
      return;
    }
    setBusy(true);
    setStatus("Stamping keywords onto your original PDF…");
    try {
      const tailored = await api.tailorResume({
        ...jobPayload,
        tone,
        missing_keywords: analyze.missing_keywords,
        missing_skills: analyze.missing_skills,
        suggestions: analyze.suggestions,
      });
      setResult(tailored);
      setAnalyze(tailored.analyze || analyze);
      if (isUuid(job.id)) {
        try {
          await api.upsertApplication(job.id, "prepared");
        } catch {
          /* optional */
        }
      }
      setStatus(tailored.changes_summary || "Tailored PDF ready");
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Tailor failed");
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

  async function downloadTailoredPdf() {
    if (!result?.download_path) return;
    try {
      await downloadAuthed(
        result.download_path,
        `${job.company}-${job.title}-tailored.pdf`.replace(/\s+/g, "-"),
      );
      setStatus("Tailored PDF downloaded.");
    } catch {
      setStatus("PDF download failed — is the API running?");
    }
  }

  async function downloadOriginalPdf() {
    try {
      await downloadAuthed("/api/v1/ai/original-resume/file", "original-resume.pdf");
      setStatus("Original PDF downloaded.");
    } catch {
      setStatus("Could not download original resume.");
    }
  }

  const gaps = analyze || result?.analyze;

  return (
    <div className="container section" style={{ maxWidth: 920 }}>
      <Link href={`/jobs/${job.id}`} style={{ color: "var(--muted)", fontWeight: 700, fontSize: "0.85rem" }}>
        ← Back
      </Link>
      <h1 className="page-title" style={{ marginTop: "0.75rem" }}>
        Prepare application
      </h1>
      <p className="section-sub">
        {job.title} · {job.company}. We score your uploaded PDF with real keyword coverage, then
        stamp missing ATS keywords onto a copy of that same PDF (keeps your 1-page layout).
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
        <button type="button" className="btn btn-primary" disabled={busy || !authed} onClick={runAnalyze}>
          {busy && !analyze ? "Analyzing…" : "1. Analyze resume"}
        </button>
        <button
          type="button"
          className="btn btn-primary"
          disabled={busy || !authed || !analyze}
          onClick={runTailor}
          style={{ opacity: analyze ? 1 : 0.55 }}
        >
          {busy && analyze ? "Stamping PDF…" : "2. Tailor PDF"}
        </button>
        <button type="button" className="btn btn-ghost" disabled={!authed || busy} onClick={downloadOriginalPdf}>
          Download original PDF
        </button>
        <a href={job.source_url} target="_blank" rel="noreferrer" className="btn btn-ghost">
          Open job to apply
        </a>
      </div>

      {status ? <p className="status-pill">{status}</p> : null}

      {gaps ? (
        <section className="surface" style={{ borderRadius: 14, padding: "1.15rem", marginTop: "1.5rem" }}>
          <h2 className="section-title" style={{ fontSize: "1.15rem", margin: 0 }}>
            ATS match · {Number(gaps.match_score || 0).toFixed(0)}%
            {gaps.covered != null && gaps.total_keywords ? (
              <span style={{ fontWeight: 500, color: "var(--muted)", fontSize: "0.9rem" }}>
                {" "}
                ({gaps.covered}/{gaps.total_keywords} keywords)
              </span>
            ) : null}
          </h2>
          <p className="section-sub" style={{ marginTop: "0.5rem" }}>
            {gaps.summary}
          </p>
          <div
            style={{
              display: "grid",
              gap: "0.85rem",
              marginTop: "1rem",
              gridTemplateColumns: "repeat(auto-fit,minmax(200px,1fr))",
            }}
          >
            <div>
              <div style={{ fontSize: "0.78rem", fontWeight: 700, color: "var(--muted)" }}>Missing skills</div>
              <p style={{ margin: "0.35rem 0 0" }}>
                {(gaps.missing_skills || []).join(", ") || "—"}
              </p>
            </div>
            <div>
              <div style={{ fontSize: "0.78rem", fontWeight: 700, color: "var(--muted)" }}>Missing keywords</div>
              <p style={{ margin: "0.35rem 0 0" }}>
                {(gaps.missing_keywords || []).join(", ") || "—"}
              </p>
            </div>
            <div>
              <div style={{ fontSize: "0.78rem", fontWeight: 700, color: "var(--muted)" }}>Already on resume</div>
              <p style={{ margin: "0.35rem 0 0" }}>
                {(gaps.strengths || []).join(", ") || "—"}
              </p>
            </div>
          </div>
          {(gaps.suggestions || []).length > 0 ? (
            <ul style={{ margin: "1rem 0 0", paddingLeft: "1.1rem", color: "var(--muted)" }}>
              {gaps.suggestions.map((s) => (
                <li key={s}>{s}</li>
              ))}
            </ul>
          ) : null}
        </section>
      ) : null}

      {result ? (
        <div style={{ display: "grid", gap: "1.25rem", marginTop: "1.25rem" }}>
          <section className="surface" style={{ borderRadius: 14, padding: "1.15rem" }}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: "0.75rem", flexWrap: "wrap" }}>
              <h2 className="section-title" style={{ fontSize: "1.15rem", margin: 0 }}>
                Tailored PDF
              </h2>
              <button type="button" className="btn btn-primary btn-sm" onClick={downloadTailoredPdf}>
                Download tailored PDF
              </button>
            </div>
            <p className="section-sub" style={{ marginTop: "0.75rem" }}>
              {result.changes_summary || result.summary}
            </p>
            {(result.keywords_added || []).length > 0 ? (
              <p style={{ marginTop: "0.75rem" }}>
                <span style={{ fontSize: "0.78rem", fontWeight: 700, color: "var(--muted)" }}>
                  Keywords stamped:{" "}
                </span>
                {result.keywords_added!.join(", ")}
              </p>
            ) : null}
          </section>

          {result.cover_letter ? (
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
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
