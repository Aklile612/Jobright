import type { Job } from "@/lib/types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function loadSoftwareJobs(): Promise<Job[]> {
  // Fast path: read from Postgres (no scrape). Sync is separate / throttled on the API.
  const fromApi = await loadFromGoApi(false);
  if (fromApi.length > 0) return fromApi;

  // Empty DB only: sync once, then list again.
  const afterSync = await loadFromGoApi(true);
  if (afterSync.length > 0) return afterSync;

  const fromRemotive = await loadFromRemotive();
  if (fromRemotive.length > 0) return fromRemotive;

  return demoJobs();
}

async function loadFromGoApi(forceSync: boolean): Promise<Job[]> {
  if (forceSync) {
    try {
      await fetch(`${API_URL}/api/v1/jobs/sync`, {
        method: "POST",
        cache: "no-store",
      });
    } catch {
      // API may be offline
    }
  }

  try {
    const res = await fetch(`${API_URL}/api/v1/jobs?limit=60`, {
      // Allow Next to cache briefly; API also caches in Redis.
      next: { revalidate: 60 },
    });
    if (!res.ok) return [];
    const jobs = (await res.json()) as Job[];
    return Array.isArray(jobs) ? jobs : [];
  } catch {
    return [];
  }
}

async function loadFromRemotive(): Promise<Job[]> {
  try {
    const res = await fetch(
      "https://remotive.com/api/remote-jobs?category=software-dev",
      {
        headers: { "User-Agent": "jobright/1.0" },
        next: { revalidate: 300 },
      },
    );
    if (!res.ok) return [];
    const payload = (await res.json()) as {
      jobs?: Array<{
        id: number;
        url: string;
        title: string;
        company_name: string;
        description: string;
        candidate_required_location?: string;
        salary?: string;
        publication_date?: string;
      }>;
    };

    return (payload.jobs || [])
      .filter((j) => j.url && j.title)
      .filter((j) => {
        try {
          return /-\d+$/.test(new URL(j.url).pathname);
        } catch {
          return false;
        }
      })
      .filter((j) => isSoftwareRole(j.title, j.description || ""))
      .slice(0, 40)
      .map((j) => ({
        id: `remotive-${j.id}`,
        title: j.title,
        company: j.company_name || "Company",
        description: stripHtml(j.description || ""),
        location: j.candidate_required_location || "Remote",
        source_url: j.url,
        salary_range: j.salary || "",
        created_at: j.publication_date || new Date().toISOString(),
      }));
  } catch {
    return [];
  }
}

function stripHtml(s: string) {
  return s
    .replace(/<[^>]+>/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 8000);
}

function isSoftwareRole(title: string, description: string) {
  const hay = `${title} ${description}`.toLowerCase();
  const keys = [
    "software",
    "engineer",
    "developer",
    "frontend",
    "backend",
    "full stack",
    "fullstack",
    "devops",
    "sre",
    "platform",
    "react",
    "typescript",
    "golang",
    "python",
    "java",
    "mobile",
    "ios",
    "android",
    "qa engineer",
    "machine learning",
    "data engineer",
  ];
  return keys.some((k) => hay.includes(k));
}

function demoJobs(): Job[] {
  const now = new Date().toISOString();
  return [
    {
      id: "demo-1",
      title: "Senior Backend Engineer (Go)",
      company: "Northwind Labs",
      description:
        "Build high-throughput APIs in Go, own Postgres schemas, and ship hiring tooling integrations.",
      location: "Remote · EU",
      source_url: "https://remoteok.com/remote-jobs",
      salary_range: "$140k–$180k",
      created_at: now,
    },
    {
      id: "demo-2",
      title: "Full Stack Engineer — TypeScript",
      company: "Harbor AI",
      description:
        "Next.js product surfaces and API services for resume intelligence features.",
      location: "Remote · US",
      source_url: "https://remotive.com/remote-jobs",
      salary_range: "$130k–$165k",
      created_at: now,
    },
  ];
}
