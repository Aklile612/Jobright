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
    const app = await api.upsertApplication(job.id, "applied");
    return app;
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
      {
        type: "jobright-autofill",
        payload: fields,
      },
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
    <div className="grid min-h-[calc(100vh-4.5rem)] lg:grid-cols-[360px_1fr]">
      <aside className="border-r border-[var(--line)] bg-[rgba(255,255,255,0.75)] p-5 backdrop-blur">
        <Link href={`/jobs/${job.id}`} className="text-sm font-semibold text-[var(--accent-deep)]">
          ← {job.title}
        </Link>
        <h1 className="mt-3 font-[family-name:var(--font-display)] text-2xl font-extrabold leading-tight">
          Apply in-site
        </h1>
        <p className="mt-1 text-sm text-[var(--ink-soft)]">
          {job.company} · application opens here so you can autofill without an extension.
        </p>

        {!authed ? (
          <div className="mt-4 rounded-2xl bg-[var(--paper-2)] p-4 text-sm">
            <Link href="/login" className="font-bold text-[var(--accent-deep)]">
              Log in
            </Link>{" "}
            to load your resume fields and track this application.
          </div>
        ) : null}

        <div className="mt-5 grid gap-3">
          {FIELD_META.map((field) => (
            <div key={field.key} className="field">
              <div className="flex items-center justify-between">
                <label htmlFor={field.key}>{field.label}</label>
                <button
                  type="button"
                  className="text-xs font-bold text-[var(--accent-deep)]"
                  onClick={() => copyField(field.key)}
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

        <div className="mt-4 rounded-2xl border border-[var(--line)] bg-white p-3 text-sm">
          <p className="font-semibold">Resume</p>
          <p className="mt-1 text-[var(--ink-soft)]">
            {autofill?.has_resume
              ? autofill.resume_file_name || autofill.resume_name
              : "No resume uploaded yet"}
          </p>
          <Link href="/resumes" className="mt-2 inline-block font-bold text-[var(--accent-deep)]">
            Manage resumes
          </Link>
        </div>

        <div className="mt-4 grid gap-2">
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

        <div className="mt-4 flex gap-2 text-xs">
          <button
            type="button"
            className={`rounded-full px-3 py-1 font-bold ${
              embedMode === "proxy"
                ? "bg-[var(--accent)] text-white"
                : "bg-[var(--paper-2)]"
            }`}
            onClick={() => setEmbedMode("proxy")}
          >
            Proxied embed
          </button>
          <button
            type="button"
            className={`rounded-full px-3 py-1 font-bold ${
              embedMode === "direct"
                ? "bg-[var(--accent)] text-white"
                : "bg-[var(--paper-2)]"
            }`}
            onClick={() => setEmbedMode("direct")}
          >
            Direct URL
          </button>
        </div>

        {status ? (
          <p className="mt-3 rounded-xl bg-[var(--accent-soft)] px-3 py-2 text-sm font-semibold text-[var(--accent-deep)]">
            {status}
          </p>
        ) : null}
      </aside>

      <section className="flex min-h-[70vh] flex-col bg-[#0d1f1a]">
        <div className="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3 text-sm text-white/80">
          <div className="truncate">
            <span className="mr-2 rounded-full bg-white/10 px-2 py-0.5 text-xs font-bold">
              Live
            </span>
            {job.source_url}
          </div>
          <a
            href={job.source_url}
            target="_blank"
            rel="noreferrer"
            className="shrink-0 font-semibold text-[#9fe0c8]"
          >
            Open original
          </a>
        </div>
        {frameSrc ? (
          <iframe
            ref={iframeRef}
            title={`Apply · ${job.title}`}
            src={frameSrc}
            className="min-h-0 flex-1 bg-white"
            sandbox="allow-forms allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox"
          />
        ) : (
          <div className="grid flex-1 place-items-center text-white/70">
            No application URL for this role
          </div>
        )}
      </section>
    </div>
  );
}
