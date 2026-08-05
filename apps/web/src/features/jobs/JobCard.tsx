import Link from "next/link";
import type { Job } from "@/lib/types";

function sourceLabel(url: string) {
  try {
    const host = new URL(url).hostname.replace(/^www\./, "");
    if (host.includes("remotive")) return "Remotive";
    if (host.includes("arbeitnow")) return "Arbeitnow";
    if (host.includes("remoteok")) return "RemoteOK";
    return host.split(".")[0] || "Board";
  } catch {
    return "Board";
  }
}

export function JobCard({ job, index = 0 }: { job: Job; index?: number }) {
  return (
    <article
      className={`surface rise group rounded-[var(--radius)] p-5 transition hover:-translate-y-0.5 hover:border-[var(--accent)]`}
      style={{ animationDelay: `${Math.min(index, 12) * 40}ms` }}
    >
      <div className="mb-3 flex items-start justify-between gap-3">
        <div>
          <p className="badge mb-2">{sourceLabel(job.source_url)}</p>
          <h3 className="font-[family-name:var(--font-display)] text-xl font-bold leading-tight">
            <Link href={`/jobs/${job.id}`} className="hover:text-[var(--accent-deep)]">
              {job.title}
            </Link>
          </h3>
          <p className="mt-1 text-sm font-semibold text-[var(--ink-soft)]">
            {job.company || "Company"} · {job.location || "Remote"}
          </p>
        </div>
        {job.salary_range ? (
          <span className="shrink-0 rounded-full bg-white px-3 py-1 text-xs font-bold text-[var(--accent-deep)] ring-1 ring-[var(--line)]">
            {job.salary_range}
          </span>
        ) : null}
      </div>
      <p className="line-clamp-3 text-sm leading-relaxed text-[var(--ink-soft)]">
        {job.description}
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        <Link href={`/jobs/${job.id}`} className="btn btn-ghost text-sm">
          Details
        </Link>
        <Link href={`/jobs/${job.id}/apply`} className="btn btn-primary text-sm">
          Apply in-site
        </Link>
      </div>
    </article>
  );
}
