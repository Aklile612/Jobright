# Architecture

## Overview

- **Go API** (`services/api`): auth, jobs, applications, resumes, scraper, extension API, AI orchestration.
- **Python AI** (`services/ai`): `POST /analyze` — score, feedback, missing keywords.
- **Next.js** (`apps/web`): user UI; talks only to Go.
- **Extension** (`apps/extension`): autofill; talks only to Go `/ext/*`.
- **PostgreSQL**: single source of truth.

## Rule

Frontend and extension never call the Python service directly.
