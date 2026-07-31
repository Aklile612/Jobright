# Architecture

- **Go API** owns auth (JWT), job catalog, applications, bookmarks, resume storage, scraper, extension API.
- **Resume_forge** owns resume parsing, ATS scoring, and job-targeted CV optimization.
- **PostgreSQL** stores JobRight domain data; forge IDs are cached on local rows.
- Frontend and extension never call Resume_forge directly.
