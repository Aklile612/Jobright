"use client";

import { useEffect, useMemo, useState } from "react";
import type { Job } from "@/lib/types";
import { API_URL } from "@/lib/api";
import { JobCard } from "./JobCard";

const PAGE_SIZE = 24;

export function JobFeed({
  initialJobs,
  initialTotal,
  title = "Open software roles",
  subtitle = "Live listings from Remotive, Arbeitnow, RemoteOK, The Muse, Jobspresso, and optional Adzuna/JSearch.",
  paginated = false,
}: {
  initialJobs: Job[];
  initialTotal?: number;
  title?: string;
  subtitle?: string;
  /** When true, Load more fetches the next page from the API. */
  paginated?: boolean;
}) {
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [location, setLocation] = useState("all");
  const [jobs, setJobs] = useState<Job[]>(initialJobs);
  const [total, setTotal] = useState(initialTotal ?? initialJobs.length);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query.trim()), 300);
    return () => clearTimeout(t);
  }, [query]);

  // Server search when paginated; reset list for each query.
  useEffect(() => {
    if (!paginated) return;
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError("");
      try {
        const params = new URLSearchParams({
          limit: String(PAGE_SIZE),
          offset: "0",
        });
        if (debouncedQuery) params.set("q", debouncedQuery);
        const res = await fetch(`${API_URL}/api/v1/jobs?${params}`, {
          cache: "no-store",
        });
        if (!res.ok) throw new Error("Could not load jobs");
        const data = await res.json();
        if (cancelled) return;
        const items = Array.isArray(data?.items)
          ? (data.items as Job[])
          : Array.isArray(data)
            ? (data as Job[])
            : [];
        setJobs(items);
        setTotal(typeof data?.total === "number" ? data.total : items.length);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load jobs");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [paginated, debouncedQuery]);

  const locations = useMemo(() => {
    const set = new Set<string>();
    jobs.forEach((j) => {
      if (j.location) set.add(j.location);
    });
    return ["all", ...Array.from(set).slice(0, 12)];
  }, [jobs]);

  const filtered = useMemo(() => {
    // When paginated, search is server-side; only filter location client-side.
    const q = paginated ? "" : query.trim().toLowerCase();
    return jobs.filter((job) => {
      const matchesQuery =
        !q ||
        job.title.toLowerCase().includes(q) ||
        job.company.toLowerCase().includes(q) ||
        job.description.toLowerCase().includes(q);
      const matchesLoc =
        location === "all" ||
        job.location.toLowerCase().includes(location.toLowerCase());
      return matchesQuery && matchesLoc;
    });
  }, [jobs, query, location, paginated]);

  async function loadMore() {
    if (!paginated || loading) return;
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({
        limit: String(PAGE_SIZE),
        offset: String(jobs.length),
      });
      if (debouncedQuery) params.set("q", debouncedQuery);
      const res = await fetch(`${API_URL}/api/v1/jobs?${params}`, {
        cache: "no-store",
      });
      if (!res.ok) throw new Error("Could not load more jobs");
      const data = await res.json();
      const items = Array.isArray(data?.items)
        ? (data.items as Job[])
        : Array.isArray(data)
          ? (data as Job[])
          : [];
      setJobs((prev) => {
        const seen = new Set(prev.map((j) => j.id));
        return [...prev, ...items.filter((j) => !seen.has(j.id))];
      });
      if (typeof data?.total === "number") setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load more");
    } finally {
      setLoading(false);
    }
  }

  const hasMore = paginated && jobs.length < total;
  const shown = filtered.length;

  return (
    <section className="section" id="roles">
      <div className="container">
        <div className="section-head">
          <div>
            <p className="badge">Live boards</p>
            <h2 className="section-title">{title}</h2>
            <p className="section-sub">{subtitle}</p>
            {paginated && total > 0 ? (
              <p className="section-sub" style={{ marginTop: 8 }}>
                Showing {shown} of {total} roles
              </p>
            ) : null}
          </div>
          <div className="filters">
            <input
              className="input"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search title, stack, company"
            />
            <select
              className="select"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
            >
              {locations.map((loc) => (
                <option key={loc} value={loc}>
                  {loc === "all" ? "All locations" : loc}
                </option>
              ))}
            </select>
          </div>
        </div>

        {error ? (
          <p className="section-sub" style={{ color: "var(--danger)" }}>
            {error}
          </p>
        ) : null}

        {filtered.length === 0 && !loading ? (
          <div className="surface empty-state">
            <p className="section-title" style={{ fontSize: "1.5rem" }}>
              No roles match yet
            </p>
            <p className="section-sub" style={{ marginInline: "auto" }}>
              Start the API to sync boards, or broaden your filters.
            </p>
          </div>
        ) : (
          <>
            <div className="job-grid">
              {filtered.map((job, i) => (
                <JobCard key={job.id} job={job} index={i} />
              ))}
            </div>
            {hasMore ? (
              <div style={{ display: "flex", justifyContent: "center", marginTop: "1.75rem" }}>
                <button
                  type="button"
                  className="btn btn-primary"
                  disabled={loading}
                  onClick={loadMore}
                >
                  {loading ? "Loading…" : "Load more jobs"}
                </button>
              </div>
            ) : null}
          </>
        )}
      </div>
    </section>
  );
}
