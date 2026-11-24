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

## Live TV update, 0.6.0

Parents can create and remove local channels with ordered programming blocks. The child guide filters the current and next program for the active profile and deadline. The stream endpoint checks the exact current slot. After-current stops a live session at the program boundary. Tests cover slot rollover, unavailable programs, permission checks and stopping. External Tunarr remains separate from the native channel implementation.

## Android TV pairing update, 0.7.0

Android TV has a Leanback launcher entry, native D-pad pairing form and origin-restricted viewer. Build with Gradle 8.11.1 and JDK 17 from android-tv. CI assembles and lints the debug APK. The parent can issue a new one-time code for the existing session. Physical remote navigation, audio, captions and resume remain device acceptance items. Human-made release branding and release signing remain pending.

## Tunarr guide import, 0.8.0

The parent can load an XMLTV guide, choose a channel, review title matches and save a local programming block. This is a snapshot import. The browser test exercises file selection, matching, channel creation and approved discovery in the child guide. Live synchronization with an external server is outside the implemented integration.

## Accessibility and endings update, 0.9.0

Profiles now control text size, contrast, navigation pictures, reduced motion and optional ending music. Pictures come from approved source files. Child-facing picture and music delivery both verify the viewer session. The visual ending draws from forty facts. Removing a profile also clears its viewing session and approval assignments. Automated checks cover the picture, contrast and size settings. Actual remote, screen-reader and family usability checks still require people and devices.

## Revocation and accessibility verification, 0.9.1

Open viewer streams recheck the current session and approval while sending data. PostgreSQL streams release the initial transaction before playback and check the latest committed household state. The browser journey now scans setup, the parent dashboard, child discovery and the ending for automated accessibility violations. SECURITY.md records trust boundaries, household data retention, HTTPS setup and outstanding specialist reviews. These automated checks do not replace real-device or human testing.

## Native Android TV playback, 0.10.0

The TV viewer now opens a native Media3 player with remote controls, caption tracks and live-slot offsets. It uses only the viewer credential, refuses redirects and rechecks the session during playback. On-demand completion returns to the locked ending or selection page. Android build and lint run in CI. Native decoding, lifecycle behavior and remote controls still need a physical TV run.

## Discovery preferences and parent PWA, 0.11.0

Five optional age presets populate editable choice counts, preferred runtime and search. Parents may change those values, interests, local spoken labels and transition warnings independently. The API ranks only eligible videos and returns stable pages. The installable dashboard includes local icons and an offline connection page. Its service worker does not cache personal responses. Browser and server tests cover the profile and discovery journeys. Preset values are provisional and require family usability testing.

## Offline catalog update, 0.12.0

Directory scanning, duplicate confirmation, local NFO and video metadata sidecars, editable series/season/episode and creator/playlist metadata, missing-episode ranges and reversible overrides are implemented. The tests use synthetic local Sesame Street and Crash Course metadata records; no copyrighted episodes were bundled or used. Actual household library acceptance remains open. Remote provider enrichment and transcoding remain separate work.

## Optional provider matching, 0.13.0

Parent-initiated TVmaze series search and YouTube video ID lookup return metadata candidates. Only the query or ID goes to the selected provider. Parents choose and save a match. Provider failure leaves the local catalog intact. A selected match retains attribution and can be reverted. Tests validate outbound destinations, encoded query values, response mapping and invalid-input behavior using test responses.

## Current priority — M1 media library

Joseph selected sequential milestone completion. Earlier notes pointing to later feature work are historical. Finish the M1 library acceptance criteria in ROADMAP.md, record evidence, then advance to M2. The current probe verifies video presence and duration. Recording codec names and verifying the entire offline library journey remain actionable.

## M1 development complete · 0.14.0

Verified the starting source at 0ec7324 with Go tests, vet and the production web build. The M1 changes extend the existing FFprobe integration to save per-track codec names, add an on-demand codec refresh for older records, and display season/special and creator/playlist groups in order. Local video metadata supplies playlist positions, which parents can also edit. Reuse decision: FFprobe already exposes codec names and stream dispositions, so FamilyTV uses that existing result directly.

Verification commands: from backend, `go test ./...`, `go test -race ./...` and `go vet ./...`; from web, `npm run build` and `npx playwright test`. The browser scanned and batch-imported five synthetic season/special/playlist fixtures, checked the missing episode, verified VP8 and Opus, restored the imported title, forced a provider lookup failure, reloaded, and advanced the local playback clock. The same journey exercises approved artwork and caption settings. Go tests cover duplicate bytes, WebVTT delivery, range playback, and persistent codec enrichment of old records.

Household media: untested. Confirm the catalog against the family's own season and playlist during final testing. M1 implementation is closed for this pass. M2 remains queued.
