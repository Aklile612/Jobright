# API sketch

## Go `:8080`

- `POST /auth/signup`
- `POST /auth/login`
- `GET /jobs`
- `POST /applications`
- `POST /applications/:id/score` — Get Score flow
- `POST /resumes`
- `GET /bookmarks`
- `GET /ext/autofill-data`

## Python `:5000`

- `POST /analyze`
  - in: `{ resume_text, job_description }`
  - out: `{ score, feedback, missing_keywords }`
