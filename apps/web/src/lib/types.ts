export type User = {
  id: string;
  email: string;
  name: string;
  current_resume_id?: string | null;
};

export type Job = {
  id: string;
  title: string;
  company: string;
  description: string;
  location: string;
  source_url: string;
  salary_range: string;
  created_at: string;
};

export type Resume = {
  id: string;
  user_id: string;
  name: string;
  file_path: string;
  file_name: string;
  content_type: string;
  forge_resume_id?: string;
  created_at: string;
};

export type Application = {
  id: string;
  user_id: string;
  job_id: string;
  status: "saved" | "applied" | "interview" | "rejected" | "offer";
  match_score?: number | null;
  match_feedback?: string[];
  missing_keywords?: string[];
  forge_version_id?: string;
  created_at: string;
  updated_at: string;
  job?: Job;
};

export type AutofillData = {
  name: string;
  email: string;
  resume_id?: string;
  resume_name?: string;
  resume_file_name?: string;
  content_type?: string;
  has_resume: boolean;
  download_path?: string;
};

export type AuthResponse = {
  token: string;
  user: User;
};
