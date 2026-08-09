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
- [x] Implement XP counter per user, persisted via `/xp/{userId}` and `/xp/{userId}/add`
- [x] Derive user level from total XP (`services.LevelForXp`)
- [x] Create journal input box + submit to `/journal/{userId}` (awards bonus XP)
- [x] Migrate frontend from JS to TypeScript
- [ ] Add adventure filters (type, mood, etc.)
- [ ] Style page for mobile responsiveness

---

##  Phase 2 – Fog of War Map (COMPLETE, 2026-08-07)
- [x] Real GPS map via Leaflet + OpenStreetMap (no API key required)
- [x] "Mark as Visited" action records adventureId/lat/lng against the user
- [x] `GET /visited/{userId}` backend endpoint for the map to consume
- [x] Fog reveal: circle overlay around each visited coordinate (5km radius)
- [x] Mark visited adventures as pins on the map
- [ ] Animate fog clearing
- [ ] Show user's current region/progress summary

---

##  Phase 3 – Achievements (COMPLETE, 2026-08-07)
- [x] Compute milestones on demand from visited-adventure/journal counts
      (`services.ComputeAchievements`) — no background job, no extra persistence
- [x] `GET /achievements/{userId}` endpoint
- [ ] Add reroll limit (reset daily)
- [ ] Reward reroll tokens from achievements or journaling
- [ ] Daily quest system (e.g., "Do 1 new thing")
- [ ] Style gear badges or perks (no functionality yet)

---

##  Phase 4 – Persistent User System (COMPLETE, 2026-08-08)
- [x] Add proper user auth (session cookies + bcrypt) — replaces trusting the
      client-supplied userId from Phase 1; closes the access-control gap
      flagged in the Phase 1-3 security review
- [x] Store visited adventures and XP (done in Phase 1/2, via SQLite instead of MongoDB)
- [x] Anonymous visitors get a recorded demo (fixed real-adventure set,
      unlimited reroll); real reroll, mark-as-visited, journal, map, and
      achievements all require an account
- [ ] Load journal entries from DB for display (currently write-only from the UI)
- [ ] Let users view past adventures
- [ ] Add settings page for account and preferences

---

##  Phase 5 – Mobile & Real-World Integration
- [ ] Convert app into mobile-friendly PWA or native wrapper
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
