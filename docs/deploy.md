# Deploy notes for EthioDeploy / Render / Railway

## Why the build failed

This is a **monorepo**. The repo root has no `go.mod` or `package.json`, so Nixpacks cannot detect a language by itself.

Also pin **Go 1.22+** in `nixpacks.toml` (`go_1_22`). Older Nixpacks `go` packages fail with:

```text
cannot compile Go 1.22 code
```

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
