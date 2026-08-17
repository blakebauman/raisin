# Raisin Console

Next.js dashboard for [raisin.run](https://raisin.run) — emails, domains, API keys, webhooks, audiences, and billing.

## Dev

```bash
# from repo root
cp ../../.env.example ../../.env   # if needed
pnpm install
pnpm dev   # http://localhost:3000
```

Requires API on `NEXT_PUBLIC_API_URL` (default `http://localhost:18080`) and Postgres with Better Auth migrations (`002_better_auth.sql`).

Auth: Better Auth email/password → `POST /console/provision`, or the login page “demo” path that mints a team JWT for the seeded demo team.
