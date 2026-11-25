# FamilyTV handoff — 2026-11-23

## Child viewer update, 0.27.0

Surprise me is the only quick action above the choices. Save for later in a video’s menu defers that video and gives it priority in a future eligible pool.

## Child viewer update, 0.26.0

Child discovery has one bounded board. The session rotation varies the candidates when more fit than the profile allows. Search filters the selected choices. Older clients receive a single page.

## Child viewer update, 0.25.0

Each subject starts with a colored bookend showing the topic name and a circular picture from its first approved video. The marker stays beside that video when the grid wraps. Text-only profiles see the label alone.

## Child viewer update, 0.24.0

Saved for later, Surprise me and profile-enabled Find sit above a single-screen grid of up to 12 eligible choices. All videos and Continue watching are absent from the child UI. Parents can enable resuming a selected video from its saved position per profile. It defaults off and existing deadline checks apply. Browser tests cover phone portrait and landscape, tablet, 720p and 1080p. Issue #1 tracks physical TV acceptance.

## Child viewer update, 0.23.0

Subjects head groups on each page of eligible videos. A new session shuffles the subject order. Reopening or transferring that session keeps its order. Save for later defers an episode from the main choices until a later eligible session. Completing the video clears the save. Older favorites remain stored. Physical acceptance is tracked in issue #1.

## Child viewer update, 0.22.0

The viewer opens on approved episode artwork. Topic buttons filter the current page using subjects from parent reviews. Saved lists sit below the video choices. Find opens search when the profile allows it. The ending has a separate night screen; quiet endings omit the illustration, fact and music. Real-TV remote and household usability testing remain open in issue #1.

## App design update, 0.21.0

The parent dashboard, child viewer and Android TV pairing screen use the round type and colors from the FamilyTV homepage. The web app bundles the same licensed font locally. Profile and system reduced-motion settings suppress the one-time TV bounce. High contrast, large text and remote navigation remain available. Physical Android TV acceptance is tracked in issue #1.

## Previous checkpoint — M7, 0.20.0

Android TV supports remote code entry and phone-assisted pairing using ZXing. The TV displays a public challenge. A signed-in parent authorizes one active profile after comparing the TV identifier. The TV redeems its private one-use secret through the existing viewer exchange. Either pairing method preserves session deadlines. Media3 handles playback and audio focus.

Automated checks cover phone approval, rejected viewer authorization, replay, transfer, focus restoration, the TV-native launch link and parent stop. Physical gate: install the debug APK on an Android TV, pair by phone and remote, check D-pad focus, native decoding, captions/audio, Back and a timed stop. Record model, Android version and results. Android build/lint and desktop browser tests cannot establish this result.

## Previous checkpoint — M6, 0.19.0

Removed channel management, XMLTV import, scheduling and tuning from the server, web viewer and Android player. On upgrade, PostgreSQL schema v2 archives old lineups and retires only sessions tuned to a channel. File-backed households use the same conversion. Existing on-demand sessions, media and bookmarks are retained. Earlier live-TV entries below are historical.

Acceptance covers retired routes, a mixed-session upgrade, and the existing on-demand, accessibility and hard-deadline journeys. M7 Android TV is next. Physical TV acceptance remains open.

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
Parent-only content approval. No child-facing watch requests. On-demand playback requires an explicit choice. Server-enforced session modes including absolute Finish By deadline. Individual accessibility settings. No analytics, ads or background third-party runtime requests. Child development and learning claims must not be overstated.

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

## M2 development complete · 0.15.0

Started from main at ee2b2f5. The parent dashboard guides a new household from profile creation to video approval and pairing. Profiles can use built-in system emoji avatars, independent reading detail and a hard maximum video length. The existing shared eligibility check enforces this limit in discovery, direct playback and live programming. A preferred length still affects ranking only. Accounts created before this change keep password sign-in and profiles retain their prior defaults.

Parents can require a PIN at setup or change it in sign-in settings. A viewer link opened in a signed-in parent browser relinquishes its server-side parent session. Parents can revoke a paired screen and outstanding code without restarting the session clock. These changes reuse the existing credential digest, persisted login throttle, household transaction and viewer checks.

Verification: backend `go test ./...`, `go test -race ./...` and `go vet ./...`; web `npm run build` and `npx playwright test`. The phone-sized browser journey covers guided setup, avatar, reading detail, PIN sign-in and viewing local media. Server tests give two profiles the same approved 61-second video and verify that only the profile with the soft preference can start it. Tests also exercise child access to parent routes, revocation, persisted PIN settings and signing out a shared browser while retaining the parent's separate device.

Next milestone: M3 content approval and classification. Physical phone installation, TV remote, screen reader, caregiver usability and household-media checks are reserved for the testing pass.

Packaged Chrome identified a small touch target on the sign-in settings disclosure. Dashboard disclosure controls now have a 44-pixel minimum height and the browser journey checks that control directly.

## M3 approvals and classification · 0.16.0

Starting commit 315ab01 passed Go tests, vet and the production web build. New work extends the existing review records, profile policies, shared playback eligibility check and discovery ranking. Parents can review pending videos, approve or revoke up to 100 selected videos for one profile, filter exact series and creators and inspect the latest 200 parent review actions. Profile-level series, creator and custom category blocks take precedence over video approval. Future imports still require explicit approval. Age bands and educational fields are descriptive. Parents can record a learning goal, reviewer, source, evidence status, co-viewing tip and off-screen activity. None of these imply a learning outcome.

Coverage counts playable files by subject, creator, duration, language, accessibility, age, category and metadata source. Exclusion counts show why approved files cannot play. Suggestions can be disabled on the parent browser. Independent variety settings spread creators, subjects and series after eligibility filtering. One-creator-per-page can return fewer than twelve choices; creator cooldown ranks other creators first and retains eligible choices when there are no alternatives.

Verification from backend: `go test ./...`, `go test -race ./...`, `go vet ./...`. From web: `npm run build` and `npx playwright test`. The browser checks unapproved and revoked search, play and file access while the child stays paired, successful approval, bulk changes, coverage and review history. Server regressions cover twelve distinct creators, scarce inventory, group blocks, persistence, idempotent bulk retries and rollback. The local detached worktree used GOFLAGS=-buildvcs=false because Go's VCS metadata lookup failed there. CI builds from a standard checkout.

Next sequential milestone: M4. Household content review and assistive-device usability remain part of final testing.

## M4 child viewer and playback · 0.17.0

Started at 1556072 and verified the Go suite, vet, web build and Playwright journey before editing. Reused the existing profile-bookmark draft and the production eligibility, discovery and range-serving code. Favorites, saved videos and resume lists are profile-specific, persisted and filtered on the server. All use the same search, page size, variety settings and deadlines. Child browsing has directional focus, Back handling and a responsive grid. Surprise me picks among visible eligible choices. On-demand playback returns to choices on completion. The TV client saves progress and uses its existing Media3 player.

Reviewed Jellyfin’s EncodingHelper stream-copy decisions. FamilyTV uses its already packaged FFmpeg for remux and transcoding with local file descriptors. It retains the selected default audio track, captions, range support and the same revocation checks on generated MP4s. Conversion limits and storage are documented in README.md.

Tests: backend `go test ./...`, `go test -race ./...`, `go vet ./...`; web `npm run build`, `npx playwright test`. Generated MKV fixtures exercise H.264 copy and MPEG-4 conversion and identify the chosen audio by packet hash. Browser acceptance uses two approved videos to verify that finishing does not start the other video. It also covers favorites, saving for later, focus/Back, resume, Surprise me, range delivery and revocation. Device remote behavior, household files and caregiver usability remain for final testing. M5 is next.

## M5 session control and endings · 0.18.0

Started at d3f84f6. The parent can set a daily viewing window, total daily minutes and individual custom-category allowances per profile. New sessions take the saved settings. The server accounts elapsed session time (including pauses) and selected-category time, preserves elapsed time across pairing and restarts, and stops at the earliest applicable limit. Allowances reset on the next date in the configured IANA time zone. Midnight ends metered sessions. Finish-by selection distinguishes repeated local minutes and rejects nonexistent ones. The child viewer accounts for request latency when scheduling a deadline.

Normal endings offer a short, locally bundled fact from a parent-selected topic, with an offline source credit and optional local music. A quiet ending displays only the lockout message. Hard cutoff suppresses the sign-off. The 40 starter facts have source references; the larger corpus and classroom presentation belong to M7A.

Verification: Go tests, race detection, vet, production web build and Playwright. Synthetic sessions cover an overnight window, both daylight-saving transitions, category filtering, persisted allowance usage and transfer between two viewer contexts while a buffered video stays paused. Physical-device accessibility and family testing are pending. M6 retires the previously supported channel features.
