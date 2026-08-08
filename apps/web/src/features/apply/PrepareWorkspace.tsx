"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { api, API_URL } from "@/lib/api";
import { getToken, isAuthenticated } from "@/lib/auth";
import type { AnalyzeResult, Job, PrepareResult, TailoredVersion } from "@/lib/types";

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
  if (!res.ok) throw new Error("Download failed");
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function scoreTone(score: number) {
  if (score >= 70) return { fg: "#1f6b4a", bg: "rgba(31,107,74,0.1)", label: "Strong" };
  if (score >= 40) return { fg: "#8a5a12", bg: "rgba(138,90,18,0.12)", label: "Fair" };
  return { fg: "#8b3a3a", bg: "rgba(139,58,58,0.1)", label: "Low" };
}

function ScoreBadge({ score, sub }: { score: number; sub?: string }) {
  const tone = scoreTone(score);
  return (
    <div
      style={{
        minWidth: 64,
        height: 64,
        borderRadius: 16,
        background: tone.bg,
        color: tone.fg,
        display: "grid",
        placeItems: "center",
        fontWeight: 800,
        lineHeight: 1.05,
        flexShrink: 0,
      }}
      title={tone.label}
    >
      <div style={{ fontSize: "1.15rem" }}>{Math.round(score)}%</div>
      {sub ? <div style={{ fontSize: "0.62rem", fontWeight: 600, opacity: 0.85 }}>{sub}</div> : null}
    </div>
  );
}

function formatWhen(iso: string) {
  try {
    return new Date(iso).toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
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
  const [history, setHistory] = useState<TailoredVersion[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [historyLoading, setHistoryLoading] = useState(false);

  const jobPayload = {
    job_id: isUuid(job.id) ? job.id : undefined,
    title: job.title,
    company: job.company,
    description: job.description,
  };

  const loadHistory = useCallback(async () => {
    if (!isAuthenticated()) {
      setHistory([]);
      return;
    }
    setHistoryLoading(true);
    try {
      const res = await api.listTailoredVersions(jobPayload);
      setHistory(res.items || []);
    } catch {
      /* ignore — empty history is fine */
    } finally {
      setHistoryLoading(false);
    }
  }, [job.id, job.title, job.company, job.description]);

  useEffect(() => {
    setAuthed(isAuthenticated());
    void loadHistory();
  }, [loadHistory]);

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
      const msg = err instanceof Error ? err.message : "Analyze failed";
      setStatus(msg);
      if (msg.toLowerCase().includes("log in") || msg.toLowerCase().includes("session")) {
        setAuthed(false);
      }
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
    setStatus("Updating skills & projects on your PDF…");
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
      if (tailored.tailored_file_id) setSelectedId(tailored.tailored_file_id);
      if (isUuid(job.id)) {
        try {
          await api.upsertApplication(job.id, "prepared");
        } catch {
          /* optional */
        }
      }
      setStatus(tailored.changes_summary || "Tailored PDF ready");
      await loadHistory();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Tailor failed";
      setStatus(msg);
      if (msg.toLowerCase().includes("log in") || msg.toLowerCase().includes("session")) {
        setAuthed(false);
      }
    } finally {
      setBusy(false);
    }
  }

  async function copyCover() {
    const text = result?.cover_letter || history.find((h) => h.file_id === selectedId)?.cover_letter || "";
    if (!text) return;
    await navigator.clipboard.writeText(text);
    setStatus("Cover letter copied.");
  }

  async function downloadPath(path: string, name: string) {
    try {
      await downloadAuthed(path, name);
      setStatus("PDF downloaded.");
    } catch {
      setStatus("PDF download failed — is the API running?");
    }
  }

  const gaps = analyze || result?.analyze;
  const selected = history.find((h) => h.file_id === selectedId) || history[0] || null;
  const thisJobVersions = history.filter((h) => h.for_this_job);
  const otherVersions = history.filter((h) => !h.for_this_job);

  return (
    <div className="container section" style={{ maxWidth: 1180 }}>
      <Link href={`/jobs/${job.id}`} style={{ color: "var(--muted)", fontWeight: 700, fontSize: "0.85rem" }}>
        ← Back
      </Link>

      <div className="prepare-layout">
        <div>
          <h1 className="page-title">Prepare application</h1>
          <p className="section-sub">
            {job.title} · {job.company}. Analyze against this JD, tailor your PDF, and compare past
            versions on the right (scored for <em>this</em> job).
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
                style={{
                  opacity: tone === value ? 1 : 0.65,
                  borderColor: tone === value ? "var(--text)" : undefined,
                }}
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
              {busy && analyze ? "Updating PDF…" : "2. Tailor PDF"}
            </button>
            <button
              type="button"
              className="btn btn-ghost"
              disabled={!authed || busy}
              onClick={() => downloadPath("/api/v1/ai/original-resume/file", "original-resume.pdf")}
            >
              Original PDF
            </button>
            <a href={job.source_url} target="_blank" rel="noreferrer" className="btn btn-ghost">
              Open job
            </a>
          </div>

          {status ? <p className="status-pill">{status}</p> : null}

          {gaps ? (
            <section className="surface" style={{ borderRadius: 16, padding: "1.15rem", marginTop: "1.35rem" }}>
              <div style={{ display: "flex", gap: "1rem", alignItems: "center" }}>
                <ScoreBadge
                  score={Number(gaps.match_score || 0)}
                  sub={
                    gaps.covered != null && gaps.total_keywords
                      ? `${gaps.covered}/${gaps.total_keywords}`
                      : undefined
                  }
                />
                <div>
                  <h2 className="section-title" style={{ fontSize: "1.15rem", margin: 0 }}>
                    Current resume vs this job
                  </h2>
                  <p className="section-sub" style={{ marginTop: "0.35rem" }}>
                    {gaps.summary}
                  </p>
                </div>
              </div>
              <div
                style={{
                  display: "grid",
                  gap: "0.85rem",
                  marginTop: "1rem",
                  gridTemplateColumns: "repeat(auto-fit,minmax(180px,1fr))",
                }}
              >
                <div>
                  <div style={{ fontSize: "0.75rem", fontWeight: 700, color: "var(--muted)" }}>
                    Missing keywords
                  </div>
                  <p style={{ margin: "0.35rem 0 0", fontSize: "0.92rem" }}>
                    {(gaps.missing_keywords || []).join(", ") || "—"}
                  </p>
                </div>
                <div>
                  <div style={{ fontSize: "0.75rem", fontWeight: 700, color: "var(--muted)" }}>
                    Missing skills
                  </div>
                  <p style={{ margin: "0.35rem 0 0", fontSize: "0.92rem" }}>
                    {(gaps.missing_skills || []).join(", ") || "—"}
                  </p>
                </div>
                <div>
                  <div style={{ fontSize: "0.75rem", fontWeight: 700, color: "var(--muted)" }}>
                    Already covered
                  </div>
                  <p style={{ margin: "0.35rem 0 0", fontSize: "0.92rem" }}>
                    {(gaps.strengths || []).join(", ") || "—"}
                  </p>
                </div>
              </div>
            </section>
          ) : null}

          {result ? (
            <section className="surface" style={{ borderRadius: 16, padding: "1.15rem", marginTop: "1rem" }}>
              <div style={{ display: "flex", justifyContent: "space-between", gap: "0.75rem", flexWrap: "wrap" }}>
                <h2 className="section-title" style={{ fontSize: "1.15rem", margin: 0 }}>
                  Latest tailored PDF
                </h2>
                <div style={{ display: "flex", gap: "0.45rem" }}>
                  {result.cover_letter ? (
                    <button type="button" className="btn btn-ghost btn-sm" onClick={copyCover}>
                      Copy letter
                    </button>
                  ) : null}
                  <button
                    type="button"
                    className="btn btn-primary btn-sm"
                    onClick={() =>
                      result.download_path &&
                      downloadPath(
                        result.download_path,
                        `${job.company}-${job.title}-tailored.pdf`.replace(/\s+/g, "-"),
                      )
                    }
                  >
                    Download PDF
                  </button>
                </div>
              </div>
              <p className="section-sub" style={{ marginTop: "0.75rem" }}>
                {result.changes_summary || result.summary}
              </p>
              {(result.keywords_added || []).length > 0 ? (
                <p style={{ marginTop: "0.65rem", fontSize: "0.92rem" }}>
                  <span style={{ fontWeight: 700, color: "var(--muted)", fontSize: "0.75rem" }}>
                    Added ·{" "}
                  </span>
                  {result.keywords_added!.join(", ")}
                </p>
              ) : null}
            </section>
          ) : null}
        </div>

        <aside className="prepare-history">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", gap: "0.5rem" }}>
            <div>
              <p style={{ margin: 0, fontSize: "0.72rem", fontWeight: 700, letterSpacing: "0.04em", color: "var(--muted)", textTransform: "uppercase" }}>
                Version history
              </p>
              <h2 className="section-title" style={{ fontSize: "1.1rem", margin: "0.25rem 0 0" }}>
                Tailored PDFs
              </h2>
            </div>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              disabled={!authed || historyLoading}
              onClick={() => void loadHistory()}
            >
              Refresh
            </button>
          </div>
          <p className="section-sub" style={{ marginTop: "0.4rem", fontSize: "0.85rem" }}>
            Scores are recalculated against <strong>{job.title}</strong> at {job.company}.
          </p>

          {!authed ? (
            <p style={{ marginTop: "1rem", color: "var(--muted)" }}>Log in to see your versions.</p>
          ) : historyLoading && history.length === 0 ? (
            <p style={{ marginTop: "1rem", color: "var(--muted)" }}>Loading…</p>
          ) : history.length === 0 ? (
            <div
              style={{
                marginTop: "1rem",
                padding: "1rem",
                borderRadius: 14,
                background: "color-mix(in srgb, var(--text) 4%, transparent)",
              }}
            >
              <p style={{ margin: 0, fontWeight: 700 }}>No tailored PDFs yet</p>
              <p className="section-sub" style={{ marginTop: "0.4rem", fontSize: "0.85rem" }}>
                Analyze, then tailor — each version appears here with its ATS score for this job.
              </p>
            </div>
          ) : (
            <div style={{ display: "grid", gap: "0.75rem", marginTop: "1rem" }}>
              {thisJobVersions.length > 0 ? (
                <p style={{ margin: 0, fontSize: "0.72rem", fontWeight: 700, color: "var(--muted)", textTransform: "uppercase" }}>
                  For this job
                </p>
              ) : null}
              {thisJobVersions.map((item) => (
                <HistoryCard
                  key={item.id}
                  item={item}
                  active={selected?.file_id === item.file_id}
                  onSelect={() => setSelectedId(item.file_id)}
                  onDownload={() =>
                    downloadPath(
                      item.download_path,
                      `${item.job_company}-${item.job_title}-tailored.pdf`.replace(/\s+/g, "-"),
                    )
                  }
                />
              ))}
              {otherVersions.length > 0 ? (
                <p style={{ margin: "0.35rem 0 0", fontSize: "0.72rem", fontWeight: 700, color: "var(--muted)", textTransform: "uppercase" }}>
                  Other roles · scored for this JD
                </p>
              ) : null}
              {otherVersions.map((item) => (
                <HistoryCard
                  key={item.id}
                  item={item}
                  active={selected?.file_id === item.file_id}
                  onSelect={() => setSelectedId(item.file_id)}
                  onDownload={() =>
                    downloadPath(
                      item.download_path,
                      `${item.job_company}-${item.job_title}-tailored.pdf`.replace(/\s+/g, "-"),
                    )
                  }
                />
              ))}
            </div>
          )}

          {selected ? (
            <div
              style={{
                marginTop: "1rem",
                paddingTop: "1rem",
                borderTop: "1px solid color-mix(in srgb, var(--text) 10%, transparent)",
              }}
            >
              <p style={{ margin: 0, fontSize: "0.75rem", fontWeight: 700, color: "var(--muted)" }}>
                Selected vs this job
              </p>
              <p style={{ margin: "0.35rem 0 0", fontSize: "0.88rem" }}>
                Missing: {(selected.missing_keywords || []).slice(0, 8).join(", ") || "—"}
              </p>
              {selected.cover_letter ? (
                <button type="button" className="btn btn-ghost btn-sm" style={{ marginTop: "0.65rem" }} onClick={copyCover}>
                  Copy cover letter
                </button>
              ) : null}
            </div>
          ) : null}
        </aside>
      </div>
    </div>
  );
}

function HistoryCard({
  item,
  active,
  onSelect,
  onDownload,
}: {
  item: TailoredVersion;
  active: boolean;
  onSelect: () => void;
  onDownload: () => void;
}) {
  const tone = scoreTone(item.score_for_current_job);
  return (
    <button
      type="button"
      onClick={onSelect}
      style={{
        textAlign: "left",
        width: "100%",
        borderRadius: 14,
        border: active
          ? `1.5px solid ${tone.fg}`
          : "1px solid color-mix(in srgb, var(--text) 12%, transparent)",
        background: active ? tone.bg : "transparent",
        padding: "0.85rem",
        cursor: "pointer",
        display: "flex",
        gap: "0.75rem",
        alignItems: "flex-start",
      }}
    >
      <ScoreBadge
        score={item.score_for_current_job}
        sub={item.total_keywords ? `${item.covered}/${item.total_keywords}` : undefined}
      />
      <div style={{ minWidth: 0, flex: 1 }}>
        <div style={{ fontWeight: 700, fontSize: "0.92rem" }}>
          {item.job_title}
        </div>
        <div style={{ color: "var(--muted)", fontSize: "0.8rem", marginTop: 2 }}>
          {item.job_company} · {formatWhen(item.created_at)}
        </div>
        {(item.keywords_added || []).length > 0 ? (
          <div
            style={{
              marginTop: "0.45rem",
              fontSize: "0.75rem",
              color: "var(--muted)",
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}
          >
            + {(item.keywords_added || []).slice(0, 5).join(", ")}
          </div>
        ) : null}
        <div style={{ marginTop: "0.55rem" }}>
          <span
            role="link"
            tabIndex={0}
            className="btn btn-ghost btn-sm"
            onClick={(e) => {
              e.stopPropagation();
              onDownload();
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.stopPropagation();
                onDownload();
              }
            }}
            style={{ display: "inline-flex" }}
          >
            Download
          </span>
        </div>
      </div>
    </button>
  );
}
