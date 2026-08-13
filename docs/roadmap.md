#  Roadmap: Solo Adventure Picker

This roadmap outlines the development phases for turning the app into a gamified real-world RPG experience. It breaks features into logical chunks from MVP to mobile readiness.

---

##  Phase 0 – Foundation (COMPLETE)
- [x] Set up Go backend with routes and HTTP server
- [x] Set up Vue frontend and basic component structure
- [x] Connect frontend to backend with fetch
- [x] Style base page with CSS
- [x] Successfully reroll random adventure via button

---

##  Phase 1 – Core Loop (COMPLETE, 2026-08-07)
- [x] Design and apply animated card transition when rerolling
- [x] Display XP value for adventure types
- [x] Migrate storage from MongoDB to SQLite (no infra dependency, see backend README)
- [x] Add real lat/lng coordinates to adventures
- [x] Create and store user ID in localStorage (pre-auth)
- [x] Implement XP counter per user, persisted via `/xp` and (as of Milestone 4.5) `/visited`
- [x] Derive user level from total XP (`services.LevelForXp`)
- [x] Create journal input box + submit to `/journal/{userId}` (awards bonus XP)
- [x] Migrate frontend from JS to TypeScript
- [x] Add adventure filters (type, effort — no `mood` field exists in the
      schema/data, so filtering covers the two real columns instead)
- [x] Style page for mobile responsiveness

Note: XP counter and journal submission were API-complete here, but the
frontend never rendered XP/level anywhere until Milestone 4.5 added the HUD.

---

##  Phase 2 – Fog of War Map (COMPLETE, 2026-08-07)
- [x] Real GPS map via Leaflet + OpenStreetMap (no API key required)
- [x] "Mark as Visited" action records adventureId/lat/lng against the user
- [x] `GET /visited/{userId}` backend endpoint for the map to consume
- [x] Fog reveal: circle overlay around each visited coordinate (5km radius)
- [x] Mark visited adventures as pins on the map
- [ ] Animate fog clearing
- [x] Show user's current region/progress summary

---

##  Phase 3 – Achievements (COMPLETE, 2026-08-07)
- [x] Compute milestones on demand from visited-adventure/journal counts
      (`services.ComputeAchievements`) — no background job, no extra persistence
- [x] `GET /achievements` endpoint
- [x] Add reroll limit (reset daily) — 5 free rerolls/day via
      `users.reroll_tokens`/`reroll_reset_at`, checked atomically in `/random`
- [x] Reward reroll tokens from achievements or journaling — achievements
      are now persisted in `user_achievements` so first-time unlocks (not
      recomputation) grant tokens; journal entries grant a smaller bonus
      each time
- [x] Daily quest system ("Do 1 new thing") — completing the first new
      visited-adventure or journal entry each day marks the quest done and
      grants a reroll-token bonus; `GET /quest` reports status
- [x] Style gear badges or perks (no functionality yet) — `.hud-gear-badge`
      styling added to `HudHeader`, static placeholder, not yet earnable

Note: achievements were computable here but had no UI until Milestone 4.5's
HUD. The remaining unchecked items (reroll limits, tokens, quests) needed
the trust fixes in Milestone 4.5 first — an economy can't be built on
client-asserted XP or undeduped visits.

---

##  Phase 4 – Persistent User System (COMPLETE, 2026-08-08)
- [x] Add proper user auth (session cookies + bcrypt) — replaces trusting the
      client-supplied userId from Phase 1; closes the access-control gap
      flagged in the Phase 1-3 security review
- [x] Store visited adventures and XP (done in Phase 1/2, via SQLite instead of MongoDB)
- [x] Anonymous visitors get a recorded demo (fixed real-adventure set,
      unlimited reroll); real reroll, mark-as-visited, journal, map, and
      achievements all require an account
- [x] Load journal entries from DB for display (currently write-only from the UI)
- [x] Let users view past adventures — combined with journal entries into a
      new `/history` page
- [ ] Add settings page for account and preferences

---

##  Milestone 4.5 – Trustworthy Economy + Visible RPG (COMPLETE, 2026-08-09)
Phases 1 and 3 were API-complete but invisible: the frontend never called
`/xp` or `/achievements`, and the economy had no real foundation to build
reroll limits or tokens on top of. This milestone fixed both.

- [x] Versioned SQLite migrations (`PRAGMA user_version`) — `CREATE TABLE IF
      NOT EXISTS` alone can't alter existing databases, so schema changes
      from here on go through a migration step
- [x] `created_at` timestamps on `visited_adventures` and `journal_entries`
- [x] Unique index on `visited_adventures(user_id, adventure_id)`, with a
      migration step deduping any pre-existing double-counted visits
- [x] `POST /visited` replaces `/xp/add`: server looks up XP amount and
      coordinates from the adventure row instead of trusting the client;
      repeat visits are a no-op (`alreadyVisited` in the response) instead
      of awarding XP again
- [x] Removed the per-reroll Gemini call on `/random` — it was
      unauthenticated, synchronous, and personalized off a hardcoded
      profile that never varied; adventures now carry real seeded
      descriptions instead
- [x] `/random` requires a session (matches the "real reroll requires an
      account" rule Phase 4 already established for other actions)
- [x] Fixed a signup account-takeover bug: signup used to accept a
      client-supplied `anonymousUserId` and upsert onto it, so a second
      signup in the same browser could silently overwrite the first
      account's email/password. User IDs are now minted server-side.
- [x] Reworked the leveling curve from a hardcoded two-threshold list
      (capped at level 3) to a formula (`services.NextLevelXp`) that keeps
      scaling
- [x] XP/level/achievements HUD (`HudHeader`) on the Adventure and Map
      pages — the first UI surface for data the backend has had since
      Phase 1
- [x] Region picker on the real Adventure page (previously demo-only)

---

##  Phase 5 – Deployment
- [x] Migrate backend storage from SQLite to Postgres (Neon), fixing the
      adventure_id type mismatch (TEXT vs INTEGER) that SQLite's loose typing
      was silently masking
- [x] Replace PRAGMA user_version migration versioning with a real
      schema_migrations table (Postgres has no PRAGMA equivalent)
- [x] Make session cookies environment-aware: SameSite=None; Secure=true in
      production (required once frontend and backend are on different
      domains), SameSite=Lax unchanged for local dev
- [x] Add an env-var-driven CORS origin allowlist for production (currently
      echoes any Origin header)
- [x] Deploy backend to Vercel (Go serverless runtime)
- [x] Deploy frontend to Vercel, pointed at the deployed backend via
      VITE_API_BASE
- [x] Verify full flow (signup, reroll, mark visited, journal, achievements,
      HUD) cross-origin in production

---

##  Phase 6 – Mobile & Real-World Integration
- [x] Convert app into mobile-friendly PWA or native wrapper —
      `vite-plugin-pwa` wired in with web manifest, auto-updating service
      worker, cache-first map tiles, network-first API calls (2026-08-13)
- [ ] Use GPS to suggest nearby adventures
- [ ] Add location-based fog reveal
- [ ] Estimate travel time and duration of adventure
- [ ] Add optional cost estimator (admission, food, gas)
- [ ] Add stretch rewards for multi-day streaks

---

##  Stretch Goals / Long-Term
- Rare gear unlocks (cosmetics or perks)
- Multiple character “classes” (Explorer, Foodie, etc.)
- Guild system to join groups
- Public map sharing with journal visibility
- Monetization: cosmetics or reroll token packs
