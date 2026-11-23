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
