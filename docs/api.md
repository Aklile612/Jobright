# API

## Auth
- `POST /api/v1/auth/signup` `{ email, password, name }`
- `POST /api/v1/auth/login` `{ email, password }`
- `GET /api/v1/auth/me`

## Jobs
- `GET /api/v1/jobs?q=&limit=&offset=`
- `GET /api/v1/jobs/:id`
- `POST /api/v1/jobs` `{ title, company, description, location, source_url, salary_range }`

## Resumes
- `GET /api/v1/resumes`
- `POST /api/v1/resumes` multipart `file`, `name`
- `GET /api/v1/resumes/:id/file`
- `DELETE /api/v1/resumes/:id`

## Applications
- `GET /api/v1/applications`
- `POST /api/v1/applications` `{ job_id, status? }`
- `POST /api/v1/applications/:id/score`
- `POST /api/v1/applications/:id/forge`
- `PATCH /api/v1/applications/:id/status` `{ status }`

## Bookmarks / Extension / Scraper
- `GET|POST /api/v1/bookmarks`, `DELETE /api/v1/bookmarks/:jobId`
- `GET /api/v1/ext/autofill-data`
- `POST /api/v1/admin/scrape` `{ urls: string[] }`

Resume_forge is called by Go only (`RESUME_FORGE_URL`).
