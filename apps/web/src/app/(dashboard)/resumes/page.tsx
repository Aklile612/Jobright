"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { api } from "@/lib/api";
import { isAuthenticated } from "@/lib/auth";
import type { Resume } from "@/lib/types";

export default function ResumesPage() {
  const router = useRouter();
  const [resumes, setResumes] = useState<Resume[]>([]);
  const [name, setName] = useState("Primary resume");
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function refresh() {
    const items = await api.listResumes();
    setResumes(items);
  }

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    refresh().catch((err) => setError(err.message));
  }, [router]);

  async function onUpload(e: FormEvent) {
    e.preventDefault();
    if (!file) {
      setError("Choose a PDF or DOCX file");
      return;
    }
    setLoading(true);
    setError("");
    try {
      await api.uploadResume(file, name);
      setFile(null);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main>
      <SiteHeader solid />
      <div className="container section">
        <h1 className="page-title">Resumes</h1>
        <p className="section-sub">
          Upload once. JobRight stores the file and syncs it to Resume_forge for
          scoring and forging when you apply.
        </p>

        <form
          onSubmit={onUpload}
          className="surface"
          style={{ marginTop: "2rem", maxWidth: 520, borderRadius: 24, padding: "1.5rem", display: "grid", gap: "1rem" }}
        >
          <div className="field">
            <label htmlFor="name">Display name</label>
            <input id="name" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="field">
            <label htmlFor="file">PDF or DOCX</label>
            <input
              id="file"
              type="file"
              accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
            />
          </div>
          {error ? <p style={{ color: "var(--warn)", margin: 0 }}>{error}</p> : null}
          <button className="btn btn-primary" disabled={loading}>
            {loading ? "Uploading…" : "Upload resume"}
          </button>
        </form>

        <div style={{ display: "grid", gap: "0.75rem", marginTop: "1.75rem" }}>
          {resumes.map((resume) => (
            <div
              key={resume.id}
              className="surface"
              style={{
                display: "flex",
                flexWrap: "wrap",
                justifyContent: "space-between",
                gap: "0.75rem",
                alignItems: "center",
                borderRadius: 18,
                padding: "1rem",
              }}
            >
              <div>
                <p style={{ margin: 0, fontWeight: 700 }}>{resume.name}</p>
                <p className="section-sub" style={{ marginTop: "0.25rem" }}>
                  {resume.file_name}
                </p>
              </div>
              <button
                className="btn btn-ghost btn-sm"
                onClick={async () => {
                  await api.deleteResume(resume.id);
                  await refresh();
                }}
              >
                Delete
              </button>
            </div>
          ))}
          {resumes.length === 0 ? <p className="section-sub">No resumes yet.</p> : null}
        </div>

        <Link href="/jobs" className="btn btn-ghost" style={{ marginTop: "2rem" }}>
          Browse jobs
        </Link>
      </div>
    </main>
  );
}
