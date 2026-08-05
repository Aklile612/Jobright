"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { isAuthenticated } from "@/lib/auth";
import type { AutofillData, Job } from "@/lib/types";

type FieldKey = "name" | "email" | "phone" | "linkedin" | "github" | "website" | "coverLetter";

const FIELD_META: { key: FieldKey; label: string; multiline?: boolean }[] = [
  { key: "name", label: "Full name" },
  { key: "email", label: "Email" },
  { key: "phone", label: "Phone" },
  { key: "linkedin", label: "LinkedIn" },
  { key: "github", label: "GitHub" },
  { key: "website", label: "Portfolio" },
  { key: "coverLetter", label: "Cover letter / note", multiline: true },
];

export function ApplyWorkspace({ job }: { job: Job }) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const [authed, setAuthed] = useState(false);
  const [autofill, setAutofill] = useState<AutofillData | null>(null);
  const [status, setStatus] = useState("");
  const [embedMode, setEmbedMode] = useState<"proxy" | "direct">("proxy");
  const [fields, setFields] = useState<Record<FieldKey, string>>({
    name: "",
    email: "",
    phone: "",
    linkedin: "",
    github: "",
    website: "",
    coverLetter: "",
  });

  useEffect(() => {
    setAuthed(isAuthenticated());
    if (!isAuthenticated()) return;
    api
      .autofill()
      .then((data) => {
        setAutofill(data);
        setFields((prev) => ({
          ...prev,
          name: data.name || "",
          email: data.email || "",
        }));
      })
      .catch(() => undefined);
  }, []);

  const frameSrc = useMemo(() => {
    if (!job.source_url) return "";
    return embedMode === "proxy" ? api.proxyUrl(job.source_url) : job.source_url;
  }, [job.source_url, embedMode]);

  async function ensureApplication() {
    if (!isAuthenticated()) {
      setStatus("Log in to track this application and push autofill.");
      return null;
    }
    return api.upsertApplication(job.id, "applied");
  }

  async function copyField(key: FieldKey) {
    await navigator.clipboard.writeText(fields[key] || "");
    setStatus(`Copied ${key}`);
  }

  function pushAutofill() {
    const frame = iframeRef.current;
    if (!frame?.contentWindow) {
      setStatus("Application frame not ready");
      return;
    }
    frame.contentWindow.postMessage(
      { type: "jobright-autofill", payload: fields },
      "*",
    );
    setStatus("Autofill pushed into the application frame");
  }

  async function score() {
    try {
      const app = await ensureApplication();
      if (!app) return;
      setStatus("Scoring with Resume_forge…");
      const scored = await api.scoreApplication(app.id);
      setStatus(
        scored.match_score != null
          ? `Match score: ${scored.match_score.toFixed(0)}`
          : "Score complete",
      );
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Score failed");
    }
  }

  async function forge() {
    try {
      const app = await ensureApplication();
      if (!app) return;
      setStatus("Forging resume for this role…");
      const result = await api.forgeApplication(app.id);
      setStatus(
        result.application.match_score != null
          ? `Forged · new score ${result.application.match_score.toFixed(0)}`
          : "Resume forged",
      );
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Forge failed");
    }
  }

  return (
    <div className="apply-shell">
      <aside className="apply-aside">
        <Link href={`/jobs/${job.id}`} style={{ color: "var(--accent)", fontWeight: 700, fontSize: "0.9rem" }}>
          ← {job.title}
        </Link>
        <h1>Apply in-site</h1>
        <p className="section-sub" style={{ marginTop: "0.4rem" }}>
          {job.company} · listing opens here with autofill beside it.
        </p>

        {!authed ? (
          <div className="surface" style={{ marginTop: "1rem", padding: "0.9rem", borderRadius: 16 }}>
            <Link href="/login" style={{ color: "var(--accent)", fontWeight: 700 }}>
              Log in
            </Link>{" "}
            to load resume fields and track this application.
          </div>
        ) : null}

        <div style={{ display: "grid", gap: "0.8rem", marginTop: "1.2rem" }}>
          {FIELD_META.map((field) => (
            <div key={field.key} className="field">
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <label htmlFor={field.key}>{field.label}</label>
                <button
                  type="button"
                  onClick={() => copyField(field.key)}
                  style={{
                    background: "none",
                    border: 0,
                    color: "var(--accent)",
                    fontWeight: 700,
                    fontSize: "0.75rem",
                    cursor: "pointer",
                  }}
                >
                  Copy
                </button>
              </div>
              {field.multiline ? (
                <textarea
                  id={field.key}
                  rows={4}
                  value={fields[field.key]}
                  onChange={(e) =>
                    setFields((prev) => ({ ...prev, [field.key]: e.target.value }))
                  }
                />
              ) : (
                <input
                  id={field.key}
                  value={fields[field.key]}
                  onChange={(e) =>
                    setFields((prev) => ({ ...prev, [field.key]: e.target.value }))
                  }
                />
              )}
            </div>
          ))}
        </div>

        <div className="surface" style={{ marginTop: "1rem", padding: "0.9rem", borderRadius: 16 }}>
          <p style={{ margin: 0, fontWeight: 700 }}>Resume</p>
          <p className="section-sub" style={{ marginTop: "0.35rem" }}>
            {autofill?.has_resume
              ? autofill.resume_file_name || autofill.resume_name
              : "No resume uploaded yet"}
          </p>
          <Link href="/resumes" style={{ color: "var(--accent)", fontWeight: 700, fontSize: "0.9rem" }}>
            Manage resumes
          </Link>
        </div>

        <div style={{ display: "grid", gap: "0.55rem", marginTop: "1rem" }}>
          <button type="button" className="btn btn-primary" onClick={pushAutofill}>
            Fill application form
          </button>
          <button type="button" className="btn btn-ghost" onClick={score}>
            Get ATS score
          </button>
          <button type="button" className="btn btn-ghost" onClick={forge}>
            Forge resume for this job
          </button>
        </div>

        <div className="mode-toggle">
          <button
            type="button"
            className={embedMode === "proxy" ? "is-on" : ""}
            onClick={() => setEmbedMode("proxy")}
          >
            Proxied embed
          </button>
          <button
            type="button"
            className={embedMode === "direct" ? "is-on" : ""}
            onClick={() => setEmbedMode("direct")}
          >
            Direct URL
          </button>
        </div>

        {status ? <p className="status-pill">{status}</p> : null}
      </aside>

      <section className="apply-frame">
        <div className="apply-frame__bar">
          <div style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            <span className="badge" style={{ marginRight: "0.5rem" }}>
              Live
            </span>
            {job.source_url}
          </div>
          <a href={job.source_url} target="_blank" rel="noreferrer" style={{ color: "var(--accent)", fontWeight: 700 }}>
            Open original
          </a>
        </div>
        {frameSrc ? (
          <iframe
            ref={iframeRef}
            title={`Apply · ${job.title}`}
            src={frameSrc}
            sandbox="allow-forms allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox"
          />
        ) : (
          <div style={{ display: "grid", placeItems: "center", color: "var(--muted)", flex: 1 }}>
            No application URL for this role
          </div>
        )}
      </section>
    </div>
  );
}
