# CLAUDE.md

Guidance for Claude Code when working in this repo. Read `PLAN.md` first for
project scope and phases — this file is about *how* to write the code, not
*what* to build.

## Project context

Go web app for personal metrics tracking (product name: Compo), starting with
body composition scans and expected to grow additional feature domains over
time (food/calorie tracking
is the concrete next one). Currently one real user, but built to support
multiple accounts from the start — every table and query is scoped by
`user_id`. Optimize for simplicity and low maintenance over scale — there's
no funding for infra, so "boring and free" beats "clever and expensive," but
the data-isolation boundary between users, and the boundary between feature
domains, are not places to cut corners.

## General principles

- Prefer the standard library over third-party packages unless a package
  saves real complexity (e.g. `chi` for routing, `pgx` for Postgres). Don't
  add a dependency for something 20 lines of stdlib code can do.
- Write code for a future reader who isn't in your head right now — clear
  names over clever ones, small functions over long ones.
- No premature abstraction **within a feature**. Each feature package
  (`internal/scans/`, later `internal/nutrition/`) should be flat and
  boring internally — no interfaces, repositories, or DI containers for a
  handful of endpoints. Where abstraction *is* warranted is the boundary
  **between** features (see "Project structure" in `PLAN.md`): features
  don't import each other, don't share tables, and only share the things
  explicitly listed there (auth middleware, Supabase client, page shell).
  That boundary is what makes adding a feature later cheap, so don't
  collapse it back into one flat pile of files "to keep things simple" —
  that trades a one-time saving now for real pain the second feature you add.
- Fail loudly in development, gracefully in production. Every user-facing
  error should degrade to something usable (e.g. AI extraction fails →
  frontend falls back to manual entry, never a blank screen).

## Go specifics

- `gofmt`/`goimports` on every file, no exceptions.
- Always check and handle errors explicitly — no `_ = err`, no swallowed
  errors. Wrap with context: `fmt.Errorf("inserting scan: %w", err)`.
- Use `context.Context` for anything that hits the DB, R2, or Gemini —
  propagate request context through, set sane timeouts on outbound calls
  (the Gemini vision call especially — don't let a slow request hang).
- Struct tags for JSON should be explicit and match the frontend's field
  names exactly (see `PLAN.md` data model) — no silent renaming.
- Validate input at the handler boundary, not deep in business logic.
- Prefer table-driven tests for anything with more than two branches.

## Secrets & config

- All secrets (`DATABASE_URL`, `GEMINI_API_KEY`, R2 credentials,
  `AUTH_SECRET`) come from environment variables — never hardcoded, never
  committed. Add a `.env.example` with dummy values and keep `.env` in
  `.gitignore`.
- Fail fast on startup if a required env var is missing — don't discover it
  at request time.

## API & security

- Auth is **Supabase Auth**. Don't hand-roll password hashing, session
  tokens, or password-reset emails — that's exactly what we're using
  Supabase to avoid building. The backend's only job is to verify the
  Supabase-issued JWT on each request (via Supabase's JWKS/verification) and
  extract `user_id` from its claims.
- Every route except `/health` requires a valid Supabase JWT — check this in
  middleware, not per-handler, and attach the resolved `user_id` to the
  request context.
- **Defense in depth on data isolation**: don't rely on application code
  alone to scope queries by `user_id` — Supabase Row Level Security (RLS)
  policies on `scans` and `goals` must also enforce `user_id = auth.uid()`
  at the database layer. If the app-layer check has a bug, RLS is the
  backstop. This is the most important rule in this file — a gap here means
  one user reading or deleting another user's data.
- Same isolation rule applies to Storage: each user's photos live under a
  `user_id/`-prefixed path, with a bucket policy restricting access to that
  prefix.
- Never log secrets, JWTs, the Supabase service-role key, or full image
  payloads. The service-role key (which bypasses RLS) is server-only — never
  send it to the frontend, never log it, never use it for a request that
  should be scoped to one user.
- Validate uploaded file types/sizes before sending to Storage or Gemini
  (images only, reasonable size cap — this runs on a free tier, don't let a
  huge upload eat the request budget).
- Rate-limit `/api/extract` **per user**, not just globally — one user
  shouldn't be able to burn through the shared Gemini daily quota and lock
  everyone else out.

## Database

- All schema changes go through migration files (Supabase CLI migrations or
  plain SQL files applied in order), never manual `ALTER TABLE` against the
  dashboard. Keep migrations numbered and idempotent-safe.
- Use parameterized queries always — no string-built SQL, ever.
- Any new table holding user data needs an RLS policy in the same migration
  that creates it — don't add the table now and "add RLS later."
- Each feature's tables are its own — `scans`/`goals` belong to the scans
  feature, a future `meals`/`foods` would belong to nutrition. Don't add
  foreign keys between two features' tables unless there's an explicit,
  agreed reason; features relating to each other is a deliberate design
  decision, not something that happens by accident because the tables were
  nearby.

## Frontend

- Shared shell (`web/index.html` + a shared `fetch`/auth-token helper) is
  common infrastructure — treat it the same way as the backend's shared
  packages, don't let one feature's assumptions leak into it.
- Each feature owns one tab and one JS module (`web/scans.js`, later
  `web/nutrition.js`). Don't reach into another feature's module or DOM —
  if two features need to share UI (e.g. a combined dashboard later), that's
  a new, explicit shared component, not a cross-import.
- Keep the existing vanilla HTML/CSS/Chart.js approach from the prototype —
  no framework needed for this scope. Don't introduce React/Vue/build
  tooling unless the app's complexity genuinely outgrows plain JS.
- Keep it mobile-first — this gets used on a phone right after a gym scan.
- Manual entry must always work even if the AI extraction endpoint is down
  or errors — never make the AI path a hard dependency.

## Keep-alive job

- A GitHub Actions workflow (`.github/workflows/keep-alive.yml`) pings
  Supabase on a schedule to prevent the free tier's 7-day inactivity pause.
  Don't remove or "optimize away" this workflow — it's small but load-bearing.
- If it ever needs changing, keep the ping interval comfortably under 7 days
  (3 days gives safe margin) and make it retry on failure rather than
  silently no-op.

## Git & commits

- Small, focused commits with clear messages describing *why*, not just
  *what*.
- Don't commit `.env`, `/bin`, or any build artifacts.

## When in doubt

Re-read `PLAN.md`. If a decision isn't covered there or here, default to
the simplest option that keeps the app at $0/month and easy for one person
to maintain.
