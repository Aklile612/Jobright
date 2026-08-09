import type { Job } from "@/lib/types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export type JobsPage = {
  items: Job[];
  total: number;
  limit: number;
  offset: number;
};

export async function loadSoftwareJobs(limit = 24, offset = 0): Promise<Job[]> {
  const page = await loadJobsPage("", limit, offset, false);
  if (page.items.length > 0 || page.total > 0) return page.items;

  // Empty DB only: sync once, then list again.
  const afterSync = await loadJobsPage("", limit, offset, true);
  if (afterSync.items.length > 0) return afterSync.items;

  const fromRemotive = await loadFromRemotive();
  if (fromRemotive.length > 0) return fromRemotive;

  return demoJobs();
}

export async function loadJobsPage(
  q = "",
  limit = 24,
  offset = 0,
  forceSync = false,
): Promise<JobsPage> {
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
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    });
    if (q.trim()) params.set("q", q.trim());
    const res = await fetch(`${API_URL}/api/v1/jobs?${params}`, {
      cache: "no-store",
    });
    if (!res.ok) return { items: [], total: 0, limit, offset };
    const data = await res.json();
    // Support both new page shape and legacy array responses.
    if (Array.isArray(data)) {
      return { items: data as Job[], total: data.length, limit, offset };
    }
    const items = Array.isArray(data?.items) ? (data.items as Job[]) : [];
    const total = typeof data?.total === "number" ? data.total : items.length;
    return { items, total, limit, offset };
  } catch {
    return { items: [], total: 0, limit, offset };
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
