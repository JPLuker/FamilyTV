# FamilyTV handoff — 2026-11-23

## Source of truth
- Repository: https://github.com/JPLuker/FamilyTV (main)
- Read: docs/AUTONOMOUS_DEVELOPMENT.md, ROADMAP.md, DECISIONS.md, NO_LIST.md, AGENTS.md and open issues.
- Follow JPLuker/InternalDesignGuidelines, especially SOP.md, WORKFLOW.md, CODING.md, DESIGN.md Part A, PRIVACY.md and SECURITY-BASELINE.md.
- The temporary FamilyTV name is not final; do not generate or select mascot art or apply the optional visual style without Joseph's approval.

## Implemented on main (not yet release-verified)
- Go HTTP API: health, setup status, first-run parent setup, login, logout, authenticated admin endpoint.
- Persisted local JSON state; PBKDF2-HMAC-SHA256 password hashing; login lockout; HttpOnly/SameSite cookies; origin checks on POST.
- Parent-managed child profiles: list, create, delete, with captions, audio-description, reduced-motion and navigation fields. Protected routes and persistence tests.
- React/TypeScript parent setup, sign-in and initial profile-management dashboard.
- Dockerfile, compose.yaml, GitHub Actions workflow, tests, accessibility spec, roadmap and project documentation.
- Autonomous-development protocol committed in docs/AUTONOMOUS_DEVELOPMENT.md.

## Verification status — do not overclaim
- A separately prepared local M0 source archive previously passed Go tests, race detector and go vet.
- **The current GitHub repository code has not been independently verified by CI or local build.** The repository version was entered via GitHub connector and subsequently changed.
- Frontend dependency installation and build were not verified. Docker image/deployment not verified.
- This is an early development scaffold, **not ready for child use or internet exposure**.

## Immediate priority
1. Fetch/clone current main and run Go tests, race detector, vet, TypeScript build, Docker build. Fix failures, commit verified fixes.
2. Audit authentication/security: password KDF correctness and performance, rate-limit persistence and per-client strategy, CSRF/origin policy, session expiration/cleanup, path handling, storage consistency, secure deployment and TLS.
3. Replace development JSON storage with PostgreSQL and migrations, with integration tests and backup/export/delete.
4. Implement the remaining M0 acceptance criteria, then move directly into M1 and onward without pausing for milestone approval.
5. Keep issues and handoff updated as verified work lands. Bump versions per internal guidelines.

## Product boundaries
Parent-only content approval. No child-facing watch requests. No on-demand autoplay; continuous playback only in live TV. Server-enforced session modes including absolute Finish By deadline. Individual accessibility settings. No analytics, ads or background third-party runtime requests. Child development and learning claims must not be overstated.

## Joseph's intervention
Not required for the current technical backlog. Branding/mascot and optional theme decisions can remain pending while work continues. Only ask for decisions that genuinely block all remaining actionable work.

## Stop rule
Do not stop after a push, milestone, report, or CI success. Continue until all independently actionable work is finished or every remaining task genuinely requires Joseph. Record exact verification evidence and blockers; do not claim completion based only on committed source.

## Verification update, 0.0.2

The source tree matched GitHub commit a34fd75. Both baseline CI jobs passed in run 45240642565. This supersedes the earlier baseline verification note. The deployment job now exercises the packaged service. See TESTING.md for its command and scope. Next, audit authentication and storage failure handling.

## Authentication update, 0.0.3

Commit a654382 passed all three jobs, including container recreation, in [run 45241166684](https://github.com/JPLuker/FamilyTV/actions/runs/45241166684). The source at a34fd75 also passed local Go tests, race checks and vet. The locked frontend built locally.

The authentication regression tests cover credential hash compatibility, origin checks on state-changing methods, restart-persistent per-client lockout, and expired-session removal. Run the suite from backend with `go test ./...`, `go test -race ./...` and `go vet ./...`. Client limits use the network peer. Clients behind one reverse proxy share its allowance. Next, implement PostgreSQL storage and recovery.

## PostgreSQL update, 0.1.0

Parent authentication landed in 6823602 and passed all three CI jobs in run 45241404440. Compose now uses PostgreSQL with a private Unix socket. The old JSON volume is mounted read-only and imported once. The integration test exercises import, four concurrent HTTP writes and restart. The deployment smoke test also verifies a dump restored to a fresh database. See docs/BACKUPS.md for recovery. Profile settings and parent export remain actionable.

## Parent dashboard update, 0.2.0

PostgreSQL landed in 58bad9a after run 45241956505 passed all three jobs. Go tests, race tests and vet also passed locally against PostgreSQL 17.12. Profile management now loads on dashboard entry, supports changing navigation, captions, audio description and reduced motion, and downloads a settings-only export. The browser regression walks through sign-in, edit, export, delete and sign-out on a phone-sized viewport. Next, build local media import and playback.

## Local media update, 0.3.0

Parent-only media import, range playback, adjacent WebVTT captions, source-change detection and catalog removal are implemented. The parent picks a relative path under a read-only mount. Removing an entry preserves its source. TestParentMedia uses FFmpeg to create a real video and FFprobe to read its duration, then checks authenticated range/caption delivery and removal. Review approval, child discovery and session enforcement next.

## Parent review update, 0.4.0

Each imported video starts unapproved. Parents review the subject, captions and described soundtrack, then select individual child profiles. Accessibility eligibility requires verified metadata for each enabled requirement. The dashboard lists library coverage gaps. Tests cover rejected anonymous changes, approval persistence, unknown metadata and revocation. The phone-sized Chromium journey now includes import, review and preview.

## Viewer and session update, 0.5.0

The parent dashboard can start, end or finish-after-current a viewing session. A one-time link exchanges for a viewer-only cookie. Catalog, selection and media routes enforce profile approval, accessibility requirements and session deadlines. The separate child page supports search, keyboard navigation, deliberate on-demand playback and a session-ending fact. Tests cover replayed links, cross-role access, hard deadlines, range access, restarts of the surrounding service and parent/child browser journeys. Next, implement scheduled live channels and the guide. Physical TV and assistive-technology testing remain open.
