"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { api } from "@/lib/api";
import { isAuthenticated } from "@/lib/auth";
import type { Application } from "@/lib/types";

export default function ApplicationsPage() {
  const router = useRouter();
  const [items, setItems] = useState<Application[]>([]);
  const [error, setError] = useState("");
  const [busyId, setBusyId] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    api
      .listApplications()
      .then(setItems)
      .catch((err) => setError(err.message));
  }, [router]);

  async function score(id: string) {
    setBusyId(id);
    try {
      const updated = await api.scoreApplication(id);
      setItems((prev) => prev.map((a) => (a.id === id ? updated : a)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Score failed");
    } finally {
      setBusyId(null);
    }
  }

  async function forge(id: string) {
    setBusyId(id);
    try {
      const result = await api.forgeApplication(id);
      setItems((prev) => prev.map((a) => (a.id === id ? result.application : a)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Forge failed");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <main>
      <SiteHeader solid />
      <div className="container section">
        <h1 className="page-title">Applications</h1>
        <p className="section-sub">
          Track status, pull ATS scores, and forge resumes per role.
        </p>
        {error ? <p style={{ color: "var(--warn)", marginTop: "1rem" }}>{error}</p> : null}

        <div style={{ display: "grid", gap: "1rem", marginTop: "2rem" }}>
          {items.map((app) => (
            <article key={app.id} className="job-card">
              <div className="job-card__top">
                <div>
                  <p className="badge" style={{ textTransform: "capitalize" }}>
                    {app.status}
                  </p>
                  <h3>{app.job?.title || "Role"}</h3>
                  <p className="job-meta">
                    {app.job?.company}
                    {app.match_score != null ? ` · score ${app.match_score.toFixed(0)}` : ""}
                  </p>
                </div>
                <div className="job-actions">
                  {app.job_id ? (
                    <Link href={`/jobs/${app.job_id}/apply`} className="btn btn-primary btn-sm">
                      Continue apply
                    </Link>
                  ) : null}
                  <button
                    className="btn btn-ghost btn-sm"
                    disabled={busyId === app.id}
                    onClick={() => score(app.id)}
                  >
                    Score
                  </button>
                  <button
                    className="btn btn-ghost btn-sm"
                    disabled={busyId === app.id}
                    onClick={() => forge(app.id)}
                  >
                    Forge
                  </button>
                </div>
              </div>
              {app.missing_keywords?.length ? (
                <p className="job-desc">
                  Missing: {app.missing_keywords.slice(0, 8).join(", ")}
                </p>
              ) : null}
            </article>
          ))}
          {items.length === 0 ? (
            <div className="surface empty-state">
              <p style={{ fontWeight: 700, margin: 0 }}>No applications yet</p>
              <Link href="/jobs" className="btn btn-primary" style={{ marginTop: "1rem" }}>
                Find a role
              </Link>
            </div>
          ) : null}
        </div>
      </div>
    </main>
  );
}
