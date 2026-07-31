# Data flow

## Get Score

1. Frontend `POST /api/v1/applications/:id/score` with JobRight JWT.
2. Go loads current resume + job from Postgres.
3. Go ensures Resume_forge session (linked at signup/login).
4. Go uploads/parses resume on forge if needed; creates/parses job on forge.
5. Go calls `POST /api/v1/ats/analyze` on Resume_forge.
6. Go saves `match_score`, feedback, keywords on `applications` and returns them.

## Forge CV

1. Frontend `POST /api/v1/applications/:id/forge`.
2. Same sync steps as score.
3. Go calls `POST /api/v1/resumes/{forge_resume_id}/optimize`.
4. Go stores final ATS + forge version id and returns optimization payload.
