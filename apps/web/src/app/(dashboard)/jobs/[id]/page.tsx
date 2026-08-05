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
      <article className="container py-10">
        <p className="badge mb-4">Role</p>
        <h1 className="max-w-3xl font-[family-name:var(--font-display)] text-4xl font-extrabold tracking-tight md:text-5xl">
          {job.title}
        </h1>
        <p className="mt-3 text-lg font-semibold text-[var(--ink-soft)]">
          {job.company} · {job.location || "Remote"}
          {job.salary_range ? ` · ${job.salary_range}` : ""}
        </p>
        <div className="mt-6 flex flex-wrap gap-3">
          <Link href={`/jobs/${job.id}/apply`} className="btn btn-primary">
            Apply in-site
          </Link>
          <a
            href={job.source_url}
            target="_blank"
            rel="noreferrer"
            className="btn btn-ghost"
          >
            Original listing
          </a>
        </div>
        <div className="surface mt-8 rounded-[24px] p-6 md:p-8">
          <h2 className="font-[family-name:var(--font-display)] text-2xl font-bold">
            Description
          </h2>
          <p className="mt-4 whitespace-pre-wrap text-[15px] leading-7 text-[var(--ink-soft)]">
            {job.description}
          </p>
        </div>
      </article>
    </main>
  );
}
