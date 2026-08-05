import { loadSoftwareJobs } from "@/lib/jobs/loadJobs";
import type { Job } from "@/lib/types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

function isUuid(id: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(
    id,
  );
}

export async function resolveJob(id: string): Promise<Job | null> {
  if (isUuid(id)) {
    try {
      const res = await fetch(`${API_URL}/api/v1/jobs/${id}`, { cache: "no-store" });
      if (res.ok) return (await res.json()) as Job;
    } catch {
      // fall through
    }
  }

  const jobs = await loadSoftwareJobs();
  return jobs.find((j) => j.id === id) || null;
}
