# Compo — Build Plan

Personal fitness metrics tracker: manually logs weight, body-fat percentage,
and skeletal muscle mass, charts trends, and tracks progress toward lean-body
goals. Single user (Ivan), fully free to run.

A mockup of the intended UI exists (Log scan / Trends / Goals screens,
hamburger+drawer on mobile, fixed sidebar on desktop) — see the design
reference used during Phase 2. It's a visual reference only, not working
code to port.

## Extensibility intent

This starts as one feature (body composition scans) but is expected to grow
into more than one (food/calorie tracking is the concrete example on the
table; others may follow — workouts, sleep, etc.). The project is structured
from day one as a feature-package backend + tab-based frontend shell so a new
domain is an *addition*, not a rewrite of the existing one. See "Project
structure" below.

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
| AI vision (photo → numbers) | **Google Gemini** (`gemini-3.8-flash`, the current free vision-capable model — Gemini 2.5 Flash was retired for new API keys during build) | Free tier, vision-capable, shared across all users — see rate-limit note below |
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
/internal/scans/           — body composition scan feature: handlers, queries, migrations
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

**scans** — fields match Evolt 360's report (the actual scanner in use); the
schema isn't tied to Evolt by name so a different scanner's data can still
map onto it later
```
id, user_id (FK → auth.users.id), date, weight, body_fat, lean_body_mass,
body_fat_mass, smm, visceral_fat, bmr, tee,
lean_left_arm, lean_right_arm, lean_trunk, lean_left_leg, lean_right_leg,
fat_left_arm, fat_right_arm, fat_trunk, fat_left_leg, fat_right_leg,
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

### Phase 1 — Backend + Supabase skeleton ✅ done
- [x] Create Supabase project, enable email/password Auth
- [x] `go mod init`, `chi` router, `pgx` pointed at Supabase's Postgres
      connection string (session pooler, IPv4-compatible)
- [x] Set up the feature-package structure (see "Project structure" above):
      `/cmd/server`, `/internal/auth`, `/internal/scans`, `/internal/platform`
- [x] Migrations for `scans`, `goals` (FK to `auth.users.id`), filed under
      `/migrations` as the scans feature's own timestamped files
- [x] Row Level Security (RLS) policies on `scans` and `goals`: a user can
      only select/insert/update/delete rows where `user_id = auth.uid()`
- [x] `internal/auth` middleware verifies the Supabase JWT via JWKS (Supabase
      has moved to signing keys instead of a shared legacy secret) on every
      protected route and extracts `user_id` from it — don't re-implement
      session logic, trust Supabase's token. Shared by every current and
      future feature.
- [x] Health check endpoint (unauthenticated)
- [x] **Keep-alive cron**: GitHub Actions workflow, `schedule: cron` every 3
      days, single HTTP ping to the Supabase project (auth health endpoint)
      — prevents the 7-day free-tier pause

Note: Supabase's key system changed since this plan was first written — it
now issues **publishable**/**secret** keys instead of legacy anon/service_role
JWTs. Env vars are `SUPABASE_PUBLISHABLE_KEY` / `SUPABASE_SECRET_KEY`.

### Phase 2 — Compact manual tracking ✅ done
- [x] Add editable profile setup for height, gender, and age
- [x] Reduce scan entry to date, weight, body-fat percentage, skeletal muscle
      mass, and optional notes
- [x] Remove image extraction, Gemini, and scan-photo storage

### Phase 3 — CRUD API ✅ done
- [x] `POST /api/scans`, `GET /api/scans`, `DELETE /api/scans/:id` — all
      scoped to `request.user_id`, never a client-supplied user id
- [x] `GET /api/goals`, `PUT /api/goals` — same scoping rule
- [x] Basic input validation (date required, numeric fields sane ranges)
- [x] Ownership check on delete (`scan.user_id == request.user_id`) before
      allowing the operation, not just filtering the list view

### Phase 4 — Frontend build ✅ done
- [x] Build the shared shell (`web/index.html`): hamburger menu opening a
      left-side drawer on mobile, fixed left sidebar on desktop; tab nav
      (Log scan / Trends / Goals, with Nutrition greyed out as "soon");
      auth/token handling; shared `fetch` helper — this is what future
      features plug into
- [x] Build `web/scans.js` as the scans feature's UI: Log scan (manual-entry
      form), Trends (Chart.js line chart + latest-scan summary card), Goals
      (target form + progress bars), and editable profile setup
- [x] Keep the three-tab layout focused on the compact manual-tracking flow
- [x] Embed static assets in the Go binary

### Phase 5 — Deploy
- [ ] Push to GitHub, connect Render, set env vars:
      `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY`, `SUPABASE_SECRET_KEY`
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
