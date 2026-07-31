# Data flow: Get Score

1. User clicks **Get Score** in Next.js.
2. Go fetches user resume + job description from PostgreSQL.
3. Go `POST`s to Python `/analyze`.
4. Python returns score + feedback + missing keywords.
5. Go saves to `applications` and returns JSON to the frontend.
