"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { isAuthenticated } from "@/lib/auth";
import type { AutofillData, Job } from "@/lib/types";

type FieldKey =
  | "name"
  | "email"
  | "phone"
  | "linkedin"
  | "github"
  | "website"
  | "coverLetter";

const FIELD_META: { key: FieldKey; label: string; multiline?: boolean }[] = [
  { key: "name", label: "Full name" },
  { key: "email", label: "Email" },
  { key: "phone", label: "Phone" },
  { key: "linkedin", label: "LinkedIn" },
  { key: "github", label: "GitHub" },
  { key: "website", label: "Website" },
  { key: "coverLetter", label: "Cover letter", multiline: true },
];

function isUuid(id: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(
    id,
  );
}

export function ApplyWorkspace({ job }: { job: Job }) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const [authed, setAuthed] = useState(false);
  const [autofill, setAutofill] = useState<AutofillData | null>(null);
  const [status, setStatus] = useState("");
  const [embedMode, setEmbedMode] = useState<"proxy" | "direct">("proxy");
  const [frameError, setFrameError] = useState(false);
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
        setFields({
          name: data.name || "",
          email: data.email || "",
          phone: data.phone || "",
          linkedin: data.linkedin || "",
          github: data.github || "",
          website: data.website || "",
          coverLetter: data.cover_letter || "",
        });
      })
      .catch(() => undefined);
  }, []);

  const frameSrc = useMemo(() => {
    if (!job.source_url) return "";
    if (embedMode === "direct") return job.source_url;
    return api.proxyUrl(job.source_url);
  }, [job.source_url, embedMode]);

  useEffect(() => {
    setFrameError(false);
  }, [frameSrc]);

  async function ensureApplication() {
    if (!isAuthenticated()) {
      setStatus("Log in to track applications and load your profile.");
      return null;
    }
    if (!isUuid(job.id)) {
      setStatus("This demo listing isn’t in the database yet. Start the API and sync jobs.");
      return null;
    }
    return api.upsertApplication(job.id, "applied");
  }

  async function copyField(key: FieldKey) {
    await navigator.clipboard.writeText(fields[key] || "");
    setStatus(`Copied ${key}`);
  }

  function pushAutofill() {
    if (embedMode === "direct") {
      setStatus("Switch to “In JobRight” mode to autofill inside the page.");
      return;
    }
    const frame = iframeRef.current;
    if (!frame?.contentWindow) {
      setStatus("Application frame not ready yet.");
      return;
    }
    frame.contentWindow.postMessage(
      { type: "jobright-autofill", payload: fields },
      "*",
    );
    setStatus("Tried to fill matching fields in the form.");
  }

  async function score() {
    try {
      const app = await ensureApplication();
      if (!app) return;
      setStatus("Scoring…");
      const scored = await api.scoreApplication(app.id);
      setStatus(
        scored.match_score != null
          ? `Match score: ${scored.match_score.toFixed(0)}`
          : "Score complete",
      );
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Score failed — is the API running?");
    }
  }

  return (
    <div className="apply-shell">
      <aside className="apply-aside">
        <Link href={`/jobs/${job.id}`} style={{ color: "var(--muted)", fontWeight: 700, fontSize: "0.85rem" }}>
          ← Back
        </Link>
        <h1>Apply in-site</h1>
        <p className="section-sub">
          {job.title} · {job.company}
        </p>

        {!authed ? (
          <div className="notice">
            <Link href="/login" style={{ fontWeight: 700, textDecoration: "underline" }}>
              Log in
            </Link>{" "}
            and set your{" "}
            <Link href="/profile" style={{ fontWeight: 700, textDecoration: "underline" }}>
              profile + resume
            </Link>{" "}
            so these fields fill automatically.
          </div>
        ) : null}

        <div style={{ display: "grid", gap: "0.7rem", marginTop: "1rem" }}>
          {FIELD_META.map((field) => (
            <div key={field.key} className="field">
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <label htmlFor={field.key}>{field.label}</label>
                <button
                  type="button"
                  onClick={() => copyField(field.key)}
                  style={{
                    border: 0,
                    background: "none",
                    color: "var(--muted)",
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
                  rows={3}
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

        <div className="notice">
          Resume:{" "}
          {autofill?.has_resume
            ? autofill.resume_file_name || autofill.resume_name
            : "none — upload on Profile"}
          <div style={{ marginTop: "0.45rem" }}>
            <Link href="/profile" style={{ fontWeight: 700, textDecoration: "underline" }}>
              Edit profile & resume
            </Link>
          </div>
        </div>

        <div style={{ display: "grid", gap: "0.45rem", marginTop: "0.85rem" }}>
          <button type="button" className="btn btn-primary" onClick={pushAutofill}>
            Autofill form
          </button>
          <button type="button" className="btn btn-ghost" onClick={score}>
            Get ATS score
          </button>
        </div>

        <div className="mode-toggle">
          <button
            type="button"
            className={embedMode === "proxy" ? "is-on" : ""}
            onClick={() => setEmbedMode("proxy")}
          >
            In JobRight
          </button>
          <button
            type="button"
            className={embedMode === "direct" ? "is-on" : ""}
            onClick={() => setEmbedMode("direct")}
          >
            Direct site
          </button>
        </div>

        {frameError ? (
          <div className="notice">
            This board blocked embedding. Use <strong>Copy</strong> on fields, or open the
            original listing.
          </div>
        ) : null}

        {status ? <p className="status-pill">{status}</p> : null}
      </aside>

      <section className="apply-frame">
        <div className="apply-frame__bar">
          <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {job.source_url}
          </span>
          <a
            href={job.source_url}
            target="_blank"
            rel="noreferrer"
            style={{ fontWeight: 700, color: "var(--text)", flexShrink: 0 }}
          >
            Open original
          </a>
        </div>
        {frameSrc ? (
          <iframe
            ref={iframeRef}
            title={`Apply · ${job.title}`}
            src={frameSrc}
            sandbox="allow-forms allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox"
            onError={() => setFrameError(true)}
          />
        ) : (
          <div style={{ display: "grid", placeItems: "center", color: "var(--muted)", flex: 1 }}>
            No application URL
          </div>
        )}
      </section>
    </div>
  );
}

