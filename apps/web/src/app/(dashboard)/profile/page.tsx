"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { api } from "@/lib/api";
import { isAuthenticated } from "@/lib/auth";
import type { Resume, User } from "@/lib/types";

export default function ProfilePage() {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [profile, setProfile] = useState<User | null>(null);
  const [resumes, setResumes] = useState<Resume[]>([]);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [uploading, setUploading] = useState(false);

  async function load() {
    const [me, list] = await Promise.all([api.getProfile(), api.listResumes()]);
    setProfile(me);
    setResumes(list);
  }

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    load().catch((err) => setError(err.message || "Failed to load profile"));
  }, [router]);

  async function onFile(file: File | null) {
    if (!file) return;
    setUploading(true);
    setError("");
    setMessage("");
    try {
      const result = await api.uploadResume(file, file.name.replace(/\.[^.]+$/, "") || "My resume");
      if (result.profile) setProfile(result.profile as User);
      if (result.resume) {
        setResumes((prev) => [result.resume as Resume, ...prev.filter((r) => r.id !== result.resume?.id)]);
      } else {
        await load();
      }
      setMessage("CV uploaded — profile fields were filled from your resume.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed — is the API running?");
    } finally {
      setUploading(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  if (!profile) {
    return (
      <main>
        <SiteHeader solid />
        <div className="container section">
          <p className="section-sub">{error || "Loading…"}</p>
        </div>
      </main>
    );
  }

  const filled = Boolean(
    profile.phone ||
      profile.linkedin ||
      profile.github ||
      profile.website ||
      profile.location ||
      profile.headline ||
      profile.cover_letter,
  );

  return (
    <main>
      <SiteHeader solid />
      <div className="container section" style={{ maxWidth: 720 }}>
        <h1 className="page-title">Profile</h1>
        <p className="section-sub">
          Upload your CV once. We extract name, links, location, headline, and experience into your profile for autofill.
        </p>

        <div
          className="surface"
          style={{
            marginTop: "1.5rem",
            borderRadius: 14,
            padding: "1rem",
            display: "flex",
            flexWrap: "wrap",
            gap: "0.75rem",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <div>
            <div style={{ fontWeight: 700 }}>Upload CV</div>
            <div style={{ color: "var(--muted)", fontSize: "0.85rem", marginTop: 2 }}>
              PDF or DOCX · auto-fills your profile
            </div>
          </div>
          <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
            <input
              ref={inputRef}
              type="file"
              accept=".pdf,.docx,application/pdf"
              style={{ maxWidth: 220, fontSize: "0.85rem" }}
              disabled={uploading}
              onChange={(e) => onFile(e.target.files?.[0] || null)}
            />
            <button
              type="button"
              className="btn btn-primary btn-sm"
              disabled={uploading}
              onClick={() => inputRef.current?.click()}
            >
              {uploading ? "Reading CV…" : "Choose file"}
            </button>
          </div>
        </div>

        {message ? <p className="status-pill">{message}</p> : null}
        {error ? <p className="status-pill" style={{ color: "var(--warn)" }}>{error}</p> : null}

        {resumes[0] ? (
          <p className="section-sub" style={{ marginTop: "0.85rem" }}>
            Current resume: <strong style={{ color: "var(--text)" }}>{resumes[0].file_name}</strong>
          </p>
        ) : null}

        <div className="surface" style={{ marginTop: "1.25rem", borderRadius: 14, padding: "1.15rem" }}>
          <div style={{ display: "flex", justifyContent: "space-between", gap: "0.75rem", alignItems: "baseline" }}>
            <h2 className="section-title" style={{ fontSize: "1.15rem", margin: 0 }}>
              Filled from your CV
            </h2>
            <span className="badge">{filled ? "Ready" : "Waiting for CV"}</span>
          </div>

          <dl
            style={{
              display: "grid",
              gap: "0.75rem",
              marginTop: "1rem",
              gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
            }}
          >
            {[
              ["Name", profile.name],
              ["Email", profile.email],
              ["Phone", profile.phone],
              ["LinkedIn", profile.linkedin],
              ["GitHub", profile.github],
              ["Website", profile.website],
              ["Location", profile.location],
              ["Headline", profile.headline],
            ].map(([label, value]) => (
              <div key={label as string}>
                <dt style={{ color: "var(--muted)", fontSize: "0.78rem", fontWeight: 700 }}>{label}</dt>
                <dd style={{ margin: "0.2rem 0 0", wordBreak: "break-word" }}>
                  {(value as string) || "—"}
                </dd>
              </div>
            ))}
          </dl>

          {profile.cover_letter ? (
            <div style={{ marginTop: "1rem" }}>
              <div style={{ color: "var(--muted)", fontSize: "0.78rem", fontWeight: 700 }}>
                Experience / summary
              </div>
              <p style={{ margin: "0.35rem 0 0", color: "var(--muted)", whiteSpace: "pre-wrap", lineHeight: 1.5 }}>
                {profile.cover_letter}
              </p>
            </div>
          ) : null}
        </div>
      </div>
    </main>
  );
}
