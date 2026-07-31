# JobRight (clone)

Monorepo scaffold.

## Services

| Path | Role | Port |
|------|------|------|
| `apps/web` | Next.js frontend | 3000 |
| `apps/extension` | Plasmo Chrome extension (MV3) | — |
| `services/api` | Go/Gin backend | 8080 |
| `services/ai` | Python/FastAPI ATS matcher | 5000 |
| `infra/docker` | Docker Compose (api + ai + postgres) | — |

## Quick start (later)

```bash
# Frontend
cd apps/web && npm run dev

# Backend / AI / DB
docker compose -f infra/docker/docker-compose.yml up
```

## Docs

- `docs/architecture.md`
- `docs/api.md`
- `docs/data-flow.md`
# Jobright
