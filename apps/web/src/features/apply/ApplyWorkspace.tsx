"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { isAuthenticated } from "@/lib/auth";
import { resolveApplyUrl } from "@/lib/jobs/resolveApplyUrl";
import type { AutofillData, Job } from "@/lib/types";

type FieldKey =
  | "name"
  | "email"
  | "phone"
  | "linkedin"
  | "github"
  | "website"
  | "skills"
  | "coverLetter";

const FIELD_META: { key: FieldKey; label: string; multiline?: boolean }[] = [
  { key: "name", label: "Full name" },
  { key: "email", label: "Email" },
  { key: "phone", label: "Phone" },
  { key: "linkedin", label: "LinkedIn" },
  { key: "github", label: "GitHub" },
  { key: "website", label: "Website" },
  { key: "skills", label: "Skills", multiline: true },
  { key: "coverLetter", label: "Cover letter", multiline: true },
];

function isUuid(id: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(
    id,
  );
}

function splitName(full: string) {
  const parts = full.trim().split(/\s+/).filter(Boolean);
  return {
    first: parts[0] || "",
    last: parts.length > 1 ? parts.slice(1).join(" ") : "",
  };
}

export function ApplyWorkspace({ job }: { job: Job }) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const fieldsRef = useRef<Record<FieldKey, string> | null>(null);
  const [authed, setAuthed] = useState(false);
  const [autofill, setAutofill] = useState<AutofillData | null>(null);
  const [status, setStatus] = useState("");
  const [embedMode, setEmbedMode] = useState<"proxy" | "direct">("proxy");
  const [frameError, setFrameError] = useState(false);
  const [filledCount, setFilledCount] = useState<number | null>(null);
  const [generating, setGenerating] = useState(false);
  const [tone, setTone] = useState<"professional" | "concise" | "enthusiastic">(
    "professional",
  );
  const [fields, setFields] = useState<Record<FieldKey, string>>({
    name: "",
    email: "",
    phone: "",
    linkedin: "",
    github: "",
    website: "",
    skills: "",
    coverLetter: "",
  });

  fieldsRef.current = fields;

  const applyUrl = useMemo(() => resolveApplyUrl(job.source_url), [job.source_url]);

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
          skills: data.skills || "",
          coverLetter: data.cover_letter || "",
        });
      })
      .catch(() =>
        setStatus("Could not load profile — log in and upload a resume on Profile."),
      );
  }, []);

  const frameSrc = useMemo(() => {
    if (!applyUrl) return "";
    if (embedMode === "direct") return applyUrl;
    return api.proxyUrl(applyUrl);
  }, [applyUrl, embedMode]);

  useEffect(() => {
    setFrameError(false);
    setFilledCount(null);
  }, [frameSrc]);

  const buildPayload = useCallback(() => {
    const f = fieldsRef.current || fields;
    const { first, last } = splitName(f.name);
    return {
      ...f,
      firstName: first,
      lastName: last,
      cover_letter: f.coverLetter,
      location: autofill?.location || "",
      headline: autofill?.headline || "",
    };
  }, [fields, autofill]);

  const pushAutofill = useCallback(
    (silent = false) => {
      if (embedMode === "direct") {
        if (!silent) {
          setStatus("Switch to “In JobRight” mode to autofill inside the page.");
        }
        return;
      }
      const frame = iframeRef.current;
      if (!frame?.contentWindow) {
        if (!silent) setStatus("Application frame not ready yet.");
        return;
      }
      const payload = buildPayload();
      const missing = ["name", "email"].filter(
        (k) => !payload[k as keyof typeof payload],
      );
      if (missing.length && !silent) {
        setStatus(`Fill ${missing.join(" & ")} in the left panel (or upload a CV on Profile), then try again.`);
      }
      frame.contentWindow.postMessage(
        { type: "jobright-autofill", payload },
        "*",
      );
      if (!silent) setStatus("Filling matching fields in the application form…");
    },
    [embedMode, buildPayload],
  );

  useEffect(() => {
    function onMessage(event: MessageEvent) {
      const data = event.data;
      if (!data || typeof data !== "object") return;
      if (data.type === "jobright-autofill-ready") {
        pushAutofill(true);
      }
      if (data.type === "jobright-autofill-result") {
        const n = Number(data.filled) || 0;
        setFilledCount(n);
        if (n > 0) {
          setStatus(`Filled ${n} field${n === 1 ? "" : "s"} on the application form.`);
        } else {
          setStatus(
            "No matching fields yet — open steps on the form (e.g. continue after email), then click Autofill again. Or use Copy + Open original.",
          );
        }
      }
    }
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [pushAutofill]);

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

  async function generateCoverLetter() {
    if (!isAuthenticated()) {
      setStatus("Log in to generate a cover letter.");
      return;
    }
    setGenerating(true);
    setStatus("Writing cover letter with Gemini…");
    try {
      const result = await api.generateCoverLetter({
        job_id: isUuid(job.id) ? job.id : undefined,
        title: job.title,
        company: job.company,
        description: job.description,
        tone,
      });
      setFields((prev) => ({ ...prev, coverLetter: result.cover_letter }));
      setStatus("Cover letter ready — edit it, then Autofill / Copy.");
    } catch (err) {
      setStatus(
        err instanceof Error
          ? err.message
          : "Cover letter failed — check GEMINI_API_KEY and API.",
      );
    } finally {
      setGenerating(false);
    }
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

  const profileReady = Boolean(fields.name && fields.email);

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

        {authed && !profileReady ? (
          <div className="notice">
            Your profile is missing name/email.{" "}
            <Link href="/profile" style={{ fontWeight: 700, textDecoration: "underline" }}>
              Upload a CV
            </Link>{" "}
            or type them below, then click Autofill.
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
          <div style={{ display: "flex", gap: "0.4rem", flexWrap: "wrap" }}>
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
                className={`btn btn-ghost btn-sm${tone === value ? " is-on" : ""}`}
                onClick={() => setTone(value)}
                style={{
                  opacity: tone === value ? 1 : 0.7,
                  borderColor: tone === value ? "var(--text)" : undefined,
                }}
              >
                {label}
              </button>
            ))}
          </div>
          <button
            type="button"
            className="btn btn-primary"
            disabled={generating}
            onClick={generateCoverLetter}
          >
            {generating ? "Writing…" : "Generate cover letter (AI)"}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => pushAutofill(false)}
          >
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

        {frameError || filledCount === 0 ? (
          <div className="notice">
            Some boards only show an email step first, or block embeds. Enter email → continue on
            the form, click <strong>Autofill</strong> again, or use <strong>Copy</strong> +{" "}
            <strong>Open original</strong>.
          </div>
        ) : null}

        {status ? <p className="status-pill">{status}</p> : null}
      </aside>

      <section className="apply-frame">
        <div className="apply-frame__bar">
          <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {applyUrl}
          </span>
          <a
            href={applyUrl}
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
            onLoad={() => {
              setTimeout(() => pushAutofill(true), 400);
              setTimeout(() => pushAutofill(true), 1500);
            }}
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
