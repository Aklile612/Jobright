import { clearToken, getToken } from "./auth";
import type {
  Application,
  AuthResponse,
  AutofillData,
  AnalyzeResult,
  Job,
  PrepareResult,
  Resume,
  User,
} from "./types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
  auth = false,
): Promise<T> {
  const headers = new Headers(options.headers || {});
  if (!(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (auth) {
    const token = getToken();
    if (token) headers.set("Authorization", `Bearer ${token}`);
  }

  const res = await fetch(`${API_URL}${path}`, { ...options, headers });
  if (res.status === 401 && auth) {
    clearToken();
  }
  if (res.status === 204) return undefined as T;

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(res.status, data.error || data.detail || "Request failed");
  }
  return data as T;
}

export const api = {
  signup: (body: { email: string; password: string; name: string }) =>
    request<AuthResponse>("/api/v1/auth/signup", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  login: (body: { email: string; password: string }) =>
    request<AuthResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  me: () => request<User>("/api/v1/auth/me", {}, true),
  getProfile: () => request<User>("/api/v1/users/me", {}, true),
  updateProfile: (body: Partial<User>) =>
    request<User>("/api/v1/users/me", {
      method: "PATCH",
      body: JSON.stringify(body),
    }, true),
  listJobs: (q = "", limit = 40) =>
    request<Job[]>(`/api/v1/jobs?q=${encodeURIComponent(q)}&limit=${limit}`),
  getJob: (id: string) => request<Job>(`/api/v1/jobs/${id}`),
  syncJobs: () =>
    request<{ results: { source: string; ingested: number; error?: string }[] }>(
      "/api/v1/jobs/sync",
      { method: "POST" },
    ),
  listResumes: () => request<Resume[]>("/api/v1/resumes", {}, true),
  uploadResume: async (file: File, name: string) => {
    const form = new FormData();
    form.append("file", file);
    form.append("name", name);
    return request<{ resume: Resume; profile: User }>(
      "/api/v1/resumes",
      { method: "POST", body: form },
      true,
    );
  },
  deleteResume: (id: string) =>
    request<void>(`/api/v1/resumes/${id}`, { method: "DELETE" }, true),
  listApplications: () => request<Application[]>("/api/v1/applications", {}, true),
  upsertApplication: (jobId: string, status?: string) =>
    request<Application>(
      "/api/v1/applications",
      {
        method: "POST",
        body: JSON.stringify({ job_id: jobId, status }),
      },
      true,
    ),
  scoreApplication: (id: string) =>
    request<Application>(`/api/v1/applications/${id}/score`, { method: "POST" }, true),
  forgeApplication: (id: string) =>
    request<{ application: Application; optimization: unknown }>(
      `/api/v1/applications/${id}/forge`,
      { method: "POST" },
      true,
    ),
  generateCoverLetter: (body: {
    job_id?: string;
    title?: string;
    company?: string;
    description?: string;
    tone?: "professional" | "concise" | "enthusiastic";
    extra?: string;
  }) =>
    request<{ cover_letter: string; tone: string; model?: string }>(
      "/api/v1/ai/cover-letter",
      { method: "POST", body: JSON.stringify(body) },
      true,
    ),
  analyzeJob: (body: {
    job_id?: string;
    title?: string;
    company?: string;
    description?: string;
  }) =>
    request<AnalyzeResult>("/api/v1/ai/analyze", {
      method: "POST",
      body: JSON.stringify(body),
    }, true),
  tailorResume: (body: {
    job_id?: string;
    title?: string;
    company?: string;
    description?: string;
    tone?: "professional" | "concise" | "enthusiastic";
    extra?: string;
    missing_keywords?: string[];
    missing_skills?: string[];
    suggestions?: string[];
  }) =>
    request<PrepareResult>("/api/v1/ai/tailor", {
      method: "POST",
      body: JSON.stringify(body),
    }, true),
  prepareApplication: (body: {
    job_id?: string;
    title?: string;
    company?: string;
    description?: string;
    tone?: "professional" | "concise" | "enthusiastic";
    extra?: string;
  }) =>
    request<PrepareResult>("/api/v1/ai/prepare", {
      method: "POST",
      body: JSON.stringify(body),
    }, true),
  aiStatus: () =>
    request<{ enabled: boolean; provider: string; model: string }>(
      "/api/v1/ai/status",
      {},
      true,
    ),
  autofill: () => request<AutofillData>("/api/v1/ext/autofill-data", {}, true),
  /** Same-origin Next proxy — always unwrap nested / mistaken proxy URLs first */
  proxyUrl: (target: string) => {
    let url = target;
    for (let i = 0; i < 5; i++) {
      try {
        const u = new URL(url, "http://localhost:3000");
        if (u.pathname.includes("/api/proxy")) {
          const inner = u.searchParams.get("url");
          if (inner) {
            url = inner;
            continue;
          }
        }
      } catch {
        /* keep */
      }
      break;
    }
    return `/api/proxy?url=${encodeURIComponent(url)}`;
  },
};

export { API_URL };
