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
| `CORS_ORIGINS` | your frontend URL (e.g. `https://….ethiodeploy.app`) |
| `PORT` | usually injected by the platform |
| `GROQ_API_KEY` / `GEMINI_API_KEY` | optional AI |
| `REDIS_URL` | leave empty (in-memory cache) |
| `UPLOAD_DIR` | `/app/storage/resumes` |

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
