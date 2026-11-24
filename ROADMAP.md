# FamilyTV roadmap

**M1 media-library development is complete.** Automated fixture acceptance passed. Household-library sign-off is reserved for the testing pass. M2 is next in sequence. Working name and mascot pending Joseph's design decisions.

- **M0 Foundation:** Go API, parent authentication, web dashboard, PostgreSQL, Docker, CI, security and backups.
- **M1 Media library:** local import, FFprobe, playback, metadata and captions.
- **M2 Profiles and accessibility:** configurable reading/navigation level, assistive input and sensory settings.
- **M3 Approvals and Library Health:** parent-only approval, metadata review, accessibility and educational coverage.
- **M4 Child viewer:** approved-content-only discovery, TV focus, no on-demand autoplay.
- **M5 Sessions:** parent stop, after-video, duration and hard finish-by time; accessible ending.
- **M6 Live TV:** channels, EPG, optional Tunarr integration.
- **M7 Android TV:** remote-first app and pairing.
- **M8 Validation:** family usability, accessibility, security and reliability.
- **M9 v1.0 release candidate:** bounded release candidate, backups and maintenance readiness.
- **M10 Final user documentation and release:** complete user guides and optional copy-and-paste AI assistance prompts; final documentation verification and v1.0 release. This is the last roadmap milestone.

## M1 acceptance

Verify the full local import journey with a season of TV episodes and a creator playlist. Check episode order, specials, missing episodes, duplicate files, codec and duration metadata, captions and artwork. Correct an incorrect provider match, restore it, and confirm that already-imported media still plays when a metadata provider is unavailable. Record test fixtures and real-library results separately in HANDOFF.md.

Scope freeze before RC. No perpetual polish cycle. Critical bugs, privacy and accessibility blockers only.

## M10 final documentation acceptance

Complete this milestone after feature development and release-candidate validation. Publish full, accurate guides for installation, setup, administration, parent and teacher workflows, Family and Classroom modes, content acquisition and screening, library organization, imports and exports, fun facts, accessibility, playback and session controls, backups, updates and troubleshooting, covering the features actually shipped.

Include an optional **Using AI to Help Manage Your Library** section with self-contained, easy copy-and-paste prompts for discovering and screening content, organizing age and subject metadata, identifying gaps, preparing classroom playlists, managing fun facts, preparing supported imports and troubleshooting. Include relevant documentation references and actual supported formats. Explain that AI suggestions require human review. This is documentation only: no built-in AI, API integration, subscription or AI infrastructure is required. FamilyTV remains fully functional without AI.

Verify the guides and prompts against the final release candidate. Publish the complete documentation before the v1.0 release. Maintain essential setup, safety and developer documentation throughout earlier milestones; defer the comprehensive end-user guides and prompt collection to this final milestone.
