# Deploy notes for EthioDeploy / Render / Railway

## Why builds fail on EthioDeploy

1. **Monorepo** — no `go.mod` at repo root, so auto-detect fails without `nixpacks.toml`.
2. **Go version** — EthioDeploy’s Nixpacks nixpkgs often has no `go_1_22` package (`undefined variable 'go_1_22'`), and plain `go` is too old (`cannot compile Go 1.22 code`).

**Fix in repo:** `nixpacks.toml` downloads **official Go 1.22.12** from `go.dev` and builds `services/api`.

## Deploy the API (recommended)

1. Create a **Web Service** from `Aklile612/Jobright`.
2. Set **Builder** to **Dockerfile** (not Nixpacks), OR keep root `Dockerfile` + `railway.toml` / `nixpacks.toml` we added.
3. **Root directory**: leave blank (repo root) so `/Dockerfile` is used.
4. Add a **Postgres** addon and set:

| Env | Value |
|-----|--------|
| `DATABASE_URL` | from Postgres addon |
| `JWT_SECRET` | long random string |
| `CORS_ORIGINS` | optional; use `*` or leave unset. Any origin is allowed unless `CORS_STRICT=true` |
| `CORS_STRICT` | set `true` only if you want an exact allowlist from `CORS_ORIGINS` |
| `PORT` | usually injected by the platform |
| `GROQ_API_KEY` / `GEMINI_API_KEY` | optional AI |
| `ADZUNA_APP_ID` / `ADZUNA_APP_KEY` | optional — Adzuna job aggregator |
| `ADZUNA_COUNTRIES` | optional — default `us,gb,de,ca` |
| `MUSE_API_KEY` | optional — The Muse (works without key; key raises limits) |
| `JSEARCH_API_KEY` or `RAPIDAPI_KEY` | optional — JSearch via RapidAPI |
| `AI_RATE_LIMIT` | optional — max AI requests per window per user (default `20`) |
| `AI_RATE_WINDOW_SEC` | optional — window seconds (default `60`) |
| `REDIS_URL` | leave empty (in-memory cache) |
| `UPLOAD_DIR` | `/app/storage/resumes` |

Resume PDF parsing needs **`poppler-utils`** (`pdftotext`). Tailoring needs **Python + pymupdf**.

`nixpacks.toml` installs a venv at `/app/.venv-pdf` during build and sets:
- `PDF_PYTHON=/app/.venv-pdf/bin/python`
- `STAMP_PDF_SCRIPT=/app/scripts/stamp_pdf.py`

After redeploy, confirm: `GET /api/v1/ai/status` (auth) should show `pdf_stamp.python` pointing at that venv. Re-upload your CV once so parse text is stored (needed for ATS %).

5. Health check path: `/health`

## Deploy the frontend (separate service)

1. New Web Service, **Root Directory** = `apps/web`
2. Build: `npm install && npm run build`
3. Start: `npm run start`
4. Env: `NEXT_PUBLIC_API_URL=https://your-api-url`

## Local Docker check

```bash
docker build -t jobright-api .
docker run --rm -p 8080:8080 -e DATABASE_URL=... -e JWT_SECRET=dev jobright-api
```
