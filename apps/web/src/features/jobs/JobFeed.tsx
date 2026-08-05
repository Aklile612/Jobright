"use client";

import { useMemo, useState } from "react";
import type { Job } from "@/lib/types";
import { JobCard } from "./JobCard";

export function JobFeed({
  initialJobs,
  title = "Open software roles",
  subtitle = "Live listings from Remotive, Arbeitnow, RemoteOK, and your JobRight catalog.",
}: {
  initialJobs: Job[];
  title?: string;
  subtitle?: string;
}) {
  const [query, setQuery] = useState("");
  const [location, setLocation] = useState("all");

  const locations = useMemo(() => {
    const set = new Set<string>();
    initialJobs.forEach((j) => {
      if (j.location) set.add(j.location);
    });
    return ["all", ...Array.from(set).slice(0, 12)];
  }, [initialJobs]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return initialJobs.filter((job) => {
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
  }, [initialJobs, query, location]);

  return (
    <section className="section" id="roles">
      <div className="container">
        <div className="section-head">
          <div>
            <p className="badge">Live boards</p>
            <h2 className="section-title">{title}</h2>
            <p className="section-sub">{subtitle}</p>
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

        {filtered.length === 0 ? (
          <div className="surface empty-state">
            <p className="section-title" style={{ fontSize: "1.5rem" }}>
              No roles match yet
            </p>
            <p className="section-sub" style={{ marginInline: "auto" }}>
              Start the API to sync boards, or broaden your filters.
            </p>
          </div>
        ) : (
          <div className="job-grid">
            {filtered.map((job, i) => (
              <JobCard key={job.id} job={job} index={i} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
