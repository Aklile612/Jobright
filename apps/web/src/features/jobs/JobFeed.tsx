"use client";

import { useMemo, useState } from "react";
import type { Job } from "@/lib/types";
import { JobCard } from "./JobCard";

export function JobFeed({
  initialJobs,
  title = "Software engineering roles",
  subtitle = "Aggregated from Remotive, Arbeitnow, RemoteOK and your JobRight catalog.",
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
    <section className="container py-10">
      <div className="mb-8 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <p className="badge mb-3">Live boards</p>
          <h2 className="font-[family-name:var(--font-display)] text-3xl font-extrabold tracking-tight md:text-4xl">
            {title}
          </h2>
          <p className="mt-2 max-w-2xl text-[var(--ink-soft)]">{subtitle}</p>
        </div>
        <div className="flex w-full flex-col gap-2 sm:flex-row md:w-auto">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search title, stack, company"
            className="min-w-[240px] rounded-full border border-[var(--line)] bg-white px-4 py-3 text-sm outline-none focus:border-[var(--accent)]"
          />
          <select
            value={location}
            onChange={(e) => setLocation(e.target.value)}
            className="rounded-full border border-[var(--line)] bg-white px-4 py-3 text-sm outline-none"
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
        <div className="surface rounded-[var(--radius)] p-10 text-center">
          <p className="font-[family-name:var(--font-display)] text-2xl font-bold">
            No roles match yet
          </p>
          <p className="mt-2 text-[var(--ink-soft)]">
            Sync boards from the API or broaden your filters.
          </p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {filtered.map((job, i) => (
            <JobCard key={job.id} job={job} index={i} />
          ))}
        </div>
      )}
    </section>
  );
}
