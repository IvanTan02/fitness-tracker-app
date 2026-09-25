# InBody Tracker — Build Plan

Personal fitness metrics tracker: logs InBody scan results (weight, body fat %,
skeletal muscle mass, visceral fat, segmental lean mass), reads numbers off a
photo of the printout via AI vision, charts trends, and tracks progress toward
weight/body-fat/SMM goals. Single user (Ivan), fully free to run.

Reference prototype: an HTML/JS artifact already built and working (Chart.js
frontend, artifact-platform storage). Port its UI/UX logic — don't redesign
from scratch.

## Extensibility intent

This starts as one feature (InBody scans) but is expected to grow into more
than one (food/calorie tracking is the concrete example on the table; others
may follow — workouts, sleep, etc.). The project is structured from day one
as a feature-package backend + tab-based frontend shell so a new domain is
an *addition*, not a rewrite of the existing one. See "Project structure"
below.

## Goals & constraints

- **$0/month to run** at current (single-user) scale.
- **Built for multiple users from day one**, even though only Ivan uses it
  right now — every table is scoped by `user_id` and auth is real accounts,
  not a shared secret, so onboarding a second user later is a signup, not a
  migration.
- Must work well on a phone (used right after gym scans).
- AI photo-reading is a nice-to-have with a manual-entry fallback that must
  always work, even if the AI call fails or is skipped entirely.

## Stack

| Concern | Choice | Free tier notes |
|---|---|---|
| Backend | Go (`chi` router) | — |
| Database + Auth + Storage | **Supabase** (Postgres, Auth, Storage all in one project) | 500MB DB, 1GB storage, 50,000 MAU, all free — one dashboard instead of three services |
| AI vision (photo → numbers) | **Google Gemini 2.5 Flash** | Free, vision-capable, 10 RPM / 250 RPD shared across all users — see rate-limit note below |
| Hosting | **Render** free web service | Sleeps after 15 min idle; ~30s cold start, acceptable for personal use |
| Frontend | Static HTML/CSS/Chart.js, embedded in the Go binary via `embed.FS` | One deployable binary, no separate frontend host |
| Auth | **Supabase Auth** (email + password, verification & reset emails included) | Battery-included — no hand-rolled bcrypt/session code needed |
| Keep-alive | GitHub Actions cron, pings Supabase every 3 days | Free tier projects pause after 7 days idle; this prevents it with ~10 lines of YAML |

**Why Supabase over Neon+R2+hand-rolled auth:** one service instead of
three, and Supabase Auth gives us signup/login/password-reset/email
verification for free instead of building and maintaining that ourselves in
Go. The trade-off is free-tier projects pause after 7 days of inactivity —
solved with the keep-alive cron job in Phase 1, so it's a non-issue in
practice.

**Scaling note:** the free tiers above (Supabase, Gemini, Render) comfortably
support a small number of users (think single digits to low tens) without
paying anything. If this ever grows past that, Gemini's shared 250 req/day
cap is the first thing that will need addressing — see Phase 6.

## Project structure

```
/cmd/server/main.go        — wiring only: router, middleware, feature registration
/internal/auth/            — Supabase JWT verification middleware (shared by all features)
/internal/scans/           — InBody scan feature: handlers, queries, migrations
/internal/nutrition/       — (future) food/calorie tracking feature — same shape as scans/
/internal/platform/        — shared infra: Supabase client, Gemini client, config
/migrations/                — SQL migrations, numbered, one feature's tables per file
/web/                        — static frontend (HTML/CSS/JS), one file per tab
```

Each feature package (`scans`, and later `nutrition`) owns its own:
- HTTP handlers, registered onto the shared router in `main.go`
- DB queries and migrations (its own tables, its own RLS policies)
- Frontend tab (`web/scans.js`, later `web/nutrition.js`) sharing one shell
  (`web/index.html`, shared `fetch` helper, shared auth token handling)

Features share: the auth middleware, the Supabase client, the overall page
shell/tab nav, and nothing else. A feature should never import another
feature's package — if scans and nutrition ever need to relate to each
other (e.g. a combined dashboard), that's a third, explicit integration
point, not a dependency between the two.

This is not "microservices" — it's one Go binary, one deploy, one repo.
It's just organized so that adding a feature means adding a folder, not
touching existing files.

## Data model

`users` is managed by Supabase Auth (`auth.users`) — we don't create our own
table for it, just reference `auth.users.id` as the FK.

**scans**
```
id, user_id (FK → auth.users.id), date, weight, body_fat, smm, visceral_fat,
lean_left_arm, lean_right_arm, lean_trunk, lean_left_leg, lean_right_leg,
notes, report_photo_path, progress_photo_path, created_at
```
(`*_photo_path` = path within the Supabase Storage bucket, not a full URL —
generate signed URLs on read)

**goals** (one row per user)
```
user_id (FK → auth.users.id, unique), weight, body_fat, smm
```

Every query that touches `scans` or `goals` must filter by the authenticated
user's `user_id` — no endpoint should ever be able to return another user's
data. Supabase Row Level Security (RLS) policies should enforce this at the
database layer, not just in application code — see Phase 1.

## Phases

### Phase 1 — Backend + Supabase skeleton
- [ ] Create Supabase project, enable email/password Auth
- [ ] `go mod init`, `chi` router, `pgx` (or `supabase-go` where useful)
      pointed at Supabase's Postgres connection string
- [ ] Set up the feature-package structure (see "Project structure" above):
      `/cmd/server`, `/internal/auth`, `/internal/scans`, `/internal/platform`
- [ ] Migrations for `scans`, `goals` (FK to `auth.users.id`), filed under
      `/migrations` as the scans feature's own numbered files
- [ ] Row Level Security (RLS) policies on `scans` and `goals`: a user can
      only select/insert/update/delete rows where `user_id = auth.uid()`
- [ ] `internal/auth` middleware verifies the Supabase JWT (from the
      `Authorization` header) on every protected route and extracts
      `user_id` from it — don't re-implement session logic, trust Supabase's
      token. This middleware is shared by every current and future feature.
- [ ] Health check endpoint (unauthenticated)
- [ ] **Keep-alive cron**: GitHub Actions workflow, `schedule: cron` every 3
      days, single HTTP ping to the Supabase project (REST or auth health
      endpoint) — prevents the 7-day free-tier pause

### Phase 2 — Storage + AI extraction
- [ ] Supabase Storage bucket for report/progress photos, with an RLS policy
      restricting each user to their own folder (e.g. path prefixed by
      `user_id/`)
- [ ] `POST /api/extract` — accepts an image, calls Gemini 2.5 Flash with the
      extraction prompt (see prototype), returns parsed JSON
- [ ] Extraction failures must degrade gracefully — return an error the
      frontend can show, never block manual entry

### Phase 3 — CRUD API
- [ ] `POST /api/scans`, `GET /api/scans`, `DELETE /api/scans/:id` — all
      scoped to `request.user_id`, never a client-supplied user id
- [ ] `GET /api/goals`, `PUT /api/goals` — same scoping rule
- [ ] Basic input validation (date required, numeric fields sane ranges)
- [ ] Ownership check on delete (`scan.user_id == request.user_id`) before
      allowing the operation, not just filtering the list view

### Phase 4 — Frontend port
- [ ] Build the shared shell (`web/index.html`): tab nav, auth/token
      handling, shared `fetch` helper — this is what future features plug
      into
- [ ] Port existing HTML/CSS/Chart.js from the artifact prototype into
      `web/scans.js` as the scans feature's tab
- [ ] Replace `claude.use("db")` / `claude.use("assets")` / `claude.use("sample")`
      calls with `fetch()` calls to the new API
- [ ] Keep the three-tab layout (Log scan / Trends / Goals) and manual-entry
      fallback exactly as in the prototype
- [ ] Embed static assets in the Go binary

### Phase 5 — Deploy
- [ ] Push to GitHub, connect Render, set env vars:
      `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`
      (server-side only, never shipped to the frontend), `DATABASE_URL`
      (Supabase's Postgres connection string), `GEMINI_API_KEY`
- [ ] Confirm HTTPS works, confirm cold-start behavior is acceptable
- [ ] Confirm the GitHub Actions keep-alive workflow is running and actually
      reaching Supabase (check the Actions tab after the first scheduled run)

### Phase 6 — Polish & multi-user headroom
- [ ] PWA manifest so it can be added to the phone home screen
- [ ] Optional: cron/uptime ping to reduce Render cold starts
- [ ] Optional: export scans as CSV
- [ ] If/when a second user actually joins: revisit the shared Gemini
      250 req/day cap (per-user daily counter + friendly "try again
      tomorrow" message is enough at small scale before paying for anything)
- [ ] Optional: simple invite-only signup (a signup code) rather than fully
      open registration, if you don't want a public app

## Open questions to confirm before/during build
- Confirm whether to proxy image uploads through the backend or upload
  straight to Supabase Storage from the browser using a short-lived client
  token (proxying is simpler and keeps the service-role key server-only —
  default to proxying unless upload volume becomes a bottleneck, which it
  won't at this scale).
- Confirm Gemini API key setup (Google AI Studio, no card).
- Decide open signup vs. invite-only for other users joining (Supabase Auth
  supports both — invite-only just means disabling public signup and adding
  users manually or via an invite link).

## Out of scope (for now)
- Native mobile app (PWA is sufficient)
- **Food/calorie tracking** — deliberately not building this yet, but the
  project structure (feature packages, migrations-per-feature, tab shell) is
  set up so it can be added as `internal/nutrition/` + `web/nutrition.js`
  without touching the scans code
- Social/shared features (comparing progress between users, etc.) — the goal
  is just "supports more than one account," not a social product
