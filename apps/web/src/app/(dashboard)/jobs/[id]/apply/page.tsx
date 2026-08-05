import { SiteHeader } from "@/components/layout/SiteHeader";
import { ApplyWorkspace } from "@/features/apply/ApplyWorkspace";
import { loadSoftwareJobs } from "@/lib/jobs/loadJobs";
import { notFound } from "next/navigation";
import type { Job } from "@/lib/types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function getJob(id: string): Promise<Job | null> {
  try {
    const res = await fetch(`${API_URL}/api/v1/jobs/${id}`, { cache: "no-store" });
    if (res.ok) return (await res.json()) as Job;
  } catch {
    // fall through
  }
  const jobs = await loadSoftwareJobs();
  return jobs.find((j) => j.id === id) || null;
}

export default async function ApplyPage({
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
      <ApplyWorkspace job={job} />
    </main>
  );
}
