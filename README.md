# JobRight

Monorepo: Next.js frontend + Go API + Resume_forge (external) for ATS/CV forging.

## Backend

```bash
# Postgres
docker compose -f infra/docker/docker-compose.yml up postgres -d

# Resume_forge must be running on :8000 (your friend's service / fork)
# https://github.com/HenokAsaye/Resume_forge  or  https://github.com/Aklile612/Resume_forge

cp .env.example .env
cd services/api && go run ./cmd/server
```

Or full stack (API + Postgres):

```bash
docker compose -f infra/docker/docker-compose.yml up --build
```

## Go API routes

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | `/api/v1/auth/signup` | no | Register (+ link Resume_forge) |
| POST | `/api/v1/auth/login` | no | Login |
| GET | `/api/v1/auth/me` | yes | Current user |
| GET | `/api/v1/jobs` | no | List jobs |
| POST | `/api/v1/jobs` | yes | Create job |
| POST | `/api/v1/resumes` | yes | Upload resume (syncs to forge) |
| POST | `/api/v1/applications` | yes | Track application |
| POST | `/api/v1/applications/:id/score` | yes | ATS score via Resume_forge |
| POST | `/api/v1/applications/:id/forge` | yes | Optimize CV via Resume_forge |
| GET | `/api/v1/ext/autofill-data` | yes | Extension autofill |
| POST | `/api/v1/admin/scrape` | yes | Scrape job URLs |

## Architecture

Next.js / Extension → Go `:8080` → PostgreSQL  
Go → Resume_forge `:8000` for parse / ATS / optimize
