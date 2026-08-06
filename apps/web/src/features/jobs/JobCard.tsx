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
      className="job-card"
      style={{ animationDelay: `${Math.min(index, 12) * 40}ms` }}
    >
      <div className="job-card__top">
        <div>
          <p className="badge">{sourceLabel(job.source_url)}</p>
          <h3>
            <Link href={`/jobs/${job.id}`}>{job.title}</Link>
          </h3>
          <p className="job-meta">
            {job.company || "Company"} · {job.location || "Remote"}
          </p>
        </div>
        {job.salary_range ? <span className="salary-chip">{job.salary_range}</span> : null}
      </div>
      <p className="job-desc">{job.description}</p>
      <div className="job-actions">
        <Link href={`/jobs/${job.id}`} className="btn btn-ghost btn-sm">
          Details
        </Link>
        <Link href={`/jobs/${job.id}/apply`} className="btn btn-primary btn-sm">
          Prepare resume
        </Link>
      </div>
    </article>
  );
}
