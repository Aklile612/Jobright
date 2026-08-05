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
      <div className="container py-10">
        <h1 className="font-[family-name:var(--font-display)] text-4xl font-extrabold">
          Resumes
        </h1>
        <p className="mt-2 max-w-2xl text-[var(--ink-soft)]">
          Upload once. JobRight stores the file and syncs it to Resume_forge for
          scoring and forging when you apply.
        </p>

        <form onSubmit={onUpload} className="surface mt-8 grid max-w-xl gap-4 rounded-[24px] p-6">
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
          {error ? <p className="text-sm text-[var(--warn)]">{error}</p> : null}
          <button className="btn btn-primary" disabled={loading}>
            {loading ? "Uploading…" : "Upload resume"}
          </button>
        </form>

        <div className="mt-8 grid gap-3">
          {resumes.map((resume) => (
            <div
              key={resume.id}
              className="surface flex flex-wrap items-center justify-between gap-3 rounded-[18px] p-4"
            >
              <div>
                <p className="font-bold">{resume.name}</p>
                <p className="text-sm text-[var(--ink-soft)]">{resume.file_name}</p>
              </div>
              <button
                className="btn btn-ghost text-sm"
                onClick={async () => {
                  await api.deleteResume(resume.id);
                  await refresh();
                }}
              >
                Delete
              </button>
            </div>
          ))}
          {resumes.length === 0 ? (
            <p className="text-[var(--ink-soft)]">No resumes yet.</p>
          ) : null}
        </div>

        <Link href="/jobs" className="btn btn-ghost mt-8 inline-flex">
          Browse jobs
        </Link>
      </div>
    </main>
  );
}
