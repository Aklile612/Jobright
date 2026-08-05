import Link from "next/link";
import { notFound } from "next/navigation";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { loadSoftwareJobs } from "@/lib/jobs/loadJobs";
import type { Job } from "@/lib/types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function getJob(id: string): Promise<Job | null> {
  try {
    const res = await fetch(`${API_URL}/api/v1/jobs/${id}`, { cache: "no-store" });
    if (res.ok) return (await res.json()) as Job;
  } catch {
    // ignore
  }
  const jobs = await loadSoftwareJobs();
  return jobs.find((j) => j.id === id) || null;
}

export default async function JobDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const job = await getJob(id);
  if (!job) notFound();

  return (
    <main>
      <SiteHeader solid />
      <article className="container section">
        <p className="badge">Role</p>
        <h1 className="page-title" style={{ maxWidth: "40rem", marginTop: "0.75rem" }}>
          {job.title}
        </h1>
        <p className="section-sub" style={{ fontSize: "1.05rem", fontWeight: 600 }}>
          {job.company} · {job.location || "Remote"}
          {job.salary_range ? ` · ${job.salary_range}` : ""}
        </p>
        <div className="hero__cta" style={{ marginTop: "1.5rem" }}>
          <Link href={`/jobs/${job.id}/apply`} className="btn btn-primary">
            Apply in-site
          </Link>
          <a href={job.source_url} target="_blank" rel="noreferrer" className="btn btn-ghost">
            Original listing
          </a>
        </div>
        <div className="surface" style={{ marginTop: "2rem", borderRadius: 24, padding: "1.75rem" }}>
          <h2 className="section-title" style={{ fontSize: "1.5rem" }}>
            Description
          </h2>
          <p
            className="section-sub"
            style={{ maxWidth: "none", whiteSpace: "pre-wrap", marginTop: "1rem" }}
          >
            {job.description}
          </p>
        </div>
      </article>
    </main>
  );
}
