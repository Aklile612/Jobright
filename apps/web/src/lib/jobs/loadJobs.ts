import type { Job } from "@/lib/types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function syncThenList(): Promise<Job[]> {
  try {
    await fetch(`${API_URL}/api/v1/jobs/sync`, {
      method: "POST",
      cache: "no-store",
    });
  } catch {
    // API may be offline; fall through to list / demo data
  }

  try {
    const res = await fetch(`${API_URL}/api/v1/jobs?limit=60`, {
      cache: "no-store",
    });
    if (res.ok) {
      const jobs = (await res.json()) as Job[];
      if (jobs.length > 0) return jobs;
    }
  } catch {
    // ignore
  }

  return demoJobs();
}

export async function loadSoftwareJobs(): Promise<Job[]> {
  return syncThenList();
}

function demoJobs(): Job[] {
  const now = new Date().toISOString();
  return [
    {
      id: "demo-1",
      title: "Senior Backend Engineer (Go)",
      company: "Northwind Labs",
      description:
        "Build high-throughput APIs in Go, own Postgres schemas, and ship hiring tooling integrations. Experience with Gin/Gorm preferred.",
      location: "Remote · EU",
      source_url: "https://remotive.com/remote-jobs/software-dev",
      salary_range: "$140k–$180k",
      created_at: now,
    },
    {
      id: "demo-2",
      title: "Full Stack Engineer — TypeScript",
      company: "Harbor AI",
      description:
        "Next.js product surfaces, FastAPI services, and resume intelligence features. Strong TypeScript and system design.",
      location: "Remote · US",
      source_url: "https://remoteok.com/",
      salary_range: "$130k–$165k",
      created_at: now,
    },
    {
      id: "demo-3",
      title: "Platform / DevOps Engineer",
      company: "Cedar Systems",
      description:
        "Kubernetes, CI/CD, observability, and developer experience for a distributed hiring platform.",
      location: "Berlin / Hybrid",
      source_url: "https://www.arbeitnow.com/",
      salary_range: "€90k–€120k",
      created_at: now,
    },
    {
      id: "demo-4",
      title: "Frontend Engineer (React)",
      company: "Lumen Hire",
      description:
        "Craft application workspaces, job feeds, and autofill UX. Care about accessibility and motion.",
      location: "Remote",
      source_url: "https://remotive.com/remote-jobs/software-dev",
      salary_range: "$120k–$150k",
      created_at: now,
    },
    {
      id: "demo-5",
      title: "Machine Learning Engineer",
      company: "ForgeMatch",
      description:
        "ATS scoring models, keyword extraction, and resume rewriting pipelines with evaluation harnesses.",
      location: "Remote · Worldwide",
      source_url: "https://remoteok.com/",
      salary_range: "$150k–$190k",
      created_at: now,
    },
    {
      id: "demo-6",
      title: "Junior Software Engineer",
      company: "Stackyard",
      description:
        "Grow across React and Go services. Mentorship-heavy team shipping job search products.",
      location: "Addis Ababa / Remote",
      source_url: "https://www.arbeitnow.com/",
      salary_range: "Competitive",
      created_at: now,
    },
  ];
}
