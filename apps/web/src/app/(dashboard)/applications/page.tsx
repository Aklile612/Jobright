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
      setItems((prev) =>
        prev.map((a) => (a.id === id ? result.application : a)),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Forge failed");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <main>
      <SiteHeader solid />
      <div className="container py-10">
        <h1 className="font-[family-name:var(--font-display)] text-4xl font-extrabold">
          Applications
        </h1>
        <p className="mt-2 text-[var(--ink-soft)]">
          Track status, pull ATS scores, and forge resumes per role.
        </p>
        {error ? <p className="mt-4 text-sm text-[var(--warn)]">{error}</p> : null}

        <div className="mt-8 grid gap-4">
          {items.map((app) => (
            <article key={app.id} className="surface rounded-[20px] p-5">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="badge mb-2 capitalize">{app.status}</p>
                  <h2 className="font-[family-name:var(--font-display)] text-2xl font-bold">
                    {app.job?.title || "Role"}
                  </h2>
                  <p className="text-sm text-[var(--ink-soft)]">
                    {app.job?.company}
                    {app.match_score != null
                      ? ` · score ${app.match_score.toFixed(0)}`
                      : ""}
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  {app.job_id ? (
                    <Link href={`/jobs/${app.job_id}/apply`} className="btn btn-primary text-sm">
                      Continue apply
                    </Link>
                  ) : null}
                  <button
                    className="btn btn-ghost text-sm"
                    disabled={busyId === app.id}
                    onClick={() => score(app.id)}
                  >
                    Score
                  </button>
                  <button
                    className="btn btn-ghost text-sm"
                    disabled={busyId === app.id}
                    onClick={() => forge(app.id)}
                  >
                    Forge
                  </button>
                </div>
              </div>
              {app.missing_keywords?.length ? (
                <p className="mt-3 text-sm text-[var(--ink-soft)]">
                  Missing: {app.missing_keywords.slice(0, 8).join(", ")}
                </p>
              ) : null}
            </article>
          ))}
          {items.length === 0 ? (
            <div className="surface rounded-[20px] p-8 text-center">
              <p className="font-bold">No applications yet</p>
              <Link href="/jobs" className="btn btn-primary mt-4 inline-flex">
                Find a role
              </Link>
            </div>
          ) : null}
        </div>
      </div>
    </main>
  );
}
