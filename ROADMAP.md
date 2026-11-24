# FamilyTV roadmap

Active milestone: **M1 — Media library.** Finish and verify M1 before advancing to M2. Continue in numerical order thereafter. Working name and mascot pending Joseph's design decisions.

- **M0 Foundation:** Go API, parent authentication, web dashboard, PostgreSQL, Docker, CI, security and backups.
- **M1 Media library:** local import, FFprobe, playback, metadata and captions.
- **M2 Profiles and accessibility:** configurable reading/navigation level, assistive input and sensory settings.
- **M3 Approvals and Library Health:** parent-only approval, metadata review, accessibility and educational coverage.
- **M4 Child viewer:** approved-content-only discovery, TV focus, no on-demand autoplay.
- **M5 Sessions:** parent stop, after-video, duration and hard finish-by time; accessible ending.
- **M6 Live TV:** channels, EPG, optional Tunarr integration.
- **M7 Android TV:** remote-first app and pairing.
- **M8 Validation:** family usability, accessibility, security and reliability.
- **M9 v1.0:** bounded release candidate, docs, backups, maintenance.

## M1 acceptance

Verify the full local import journey with a season of TV episodes and a creator playlist. Check episode order, specials, missing episodes, duplicate files, codec and duration metadata, captions and artwork. Correct an incorrect provider match, restore it, and confirm that already-imported media still plays when a metadata provider is unavailable. Record test fixtures and real-library results separately in HANDOFF.md.

Scope freeze before RC. No perpetual polish cycle. Critical bugs, privacy and accessibility blockers only.
