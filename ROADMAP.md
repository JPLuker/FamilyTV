# FamilyTV roadmap

**M1 media library, M2 profiles, M3 approvals, M4 child viewing and M5 sessions are development-complete.** Automated acceptance uses synthetic media and profiles. Household and physical-device sign-off are reserved for testing. M6 live-TV retirement is next in sequence. Working name and mascot pending Joseph's design decisions.

- **M0 Foundation:** Go API, parent authentication, web dashboard, PostgreSQL, Docker, CI, security and backups.
- **M1 Media library:** local import, FFprobe, playback, metadata and captions.
- **M2 Profiles and accessibility:** configurable reading/navigation level, assistive input and sensory settings.
- **M3 Approvals and Library Health:** parent-only approval, metadata review, accessibility and educational coverage.
- **M4 Child viewer:** approved-content-only discovery, TV focus, no on-demand autoplay.
- **M5 Sessions:** parent stop, after-video, duration and hard finish-by time; accessible ending.
- **M6 Remove Live TV:** retire existing linear channels, EPG, XMLTV/Tunarr integration and continuous live playback; preserve on-demand media, session enforcement and user data unrelated to channels.
- **M7 Android TV:** remote-first app and pairing.
- **M7A Fun facts and Classroom Mode:** shared verified fact and authentic-image library, session endings, standalone fact presentation, teacher controls, grade/subject filters and lesson playlists.
- **M7B Educational Arcade (optional, time-boxed):** up to five lightweight fact-driven formats using shared components; never block v1.0 for additional games.
- **M8 Validation:** family usability, accessibility, security and reliability.
- **M9 v1.0 release candidate:** bounded release candidate, backups and maintenance readiness.
- **M10 Final user documentation and release:** complete user guides and optional copy-and-paste AI assistance prompts; final documentation verification and v1.0 release. This is the last roadmap milestone.

## M1 acceptance

Verify the full local import journey with a season of TV episodes and a creator playlist. Check episode order, specials, missing episodes, duplicate files, codec and duration metadata, captions and artwork. Correct an incorrect provider match, restore it, and confirm that already-imported media still plays when a metadata provider is unavailable. Record test fixtures and real-library results separately in HANDOFF.md.

Scope freeze before RC. No perpetual polish cycle. Critical bugs, privacy and accessibility blockers only.

## M2 acceptance

A parent can configure an account, create two child profiles with different permissions, approve a video and pair a viewer. Verify that the same approved video can be played by one profile and denied to another, that child credentials cannot manage the household and that revoking one viewer preserves the other profile’s session. See HANDOFF.md for the automated evidence and physical-device testing boundary.

## M3 acceptance

Verify the full parent review flow, including an unapproved search and direct playback attempt from a paired child browser. Approve, revoke while paired and check the same paths again. Group and category restrictions override individual approval. Library coverage counts only currently playable approved files for the selected profile. Verify that optional creator variety and cooldown preserve hard restrictions and handle a small eligible library.

## M4 acceptance

A paired child can browse with a D-pad, select and return to the same tile, save favorites and videos for later, resume a partially watched video and choose a surprise from eligible videos. Completing an on-demand video returns to choices with playback stopped. Approval, accessibility and deadline checks apply to shelves, subtitles, previews, direct playback and compatibility copies.

## M5 acceptance

Verify parent stop, finish-after-current, fixed duration and absolute cutoff. Pause a buffered video and move a timed session to another device. The original deadline still ends playback and locks both devices. Verify persisted daily and category accounting, daylight-saving gaps and repeated local minutes, a quiet ending and an optional short sourced fact and music.

## M6 scope change and acceptance

Live TV is explicitly out of scope for FamilyTV v1.0. Remove existing channel management, guide UI, live playback routes and XMLTV/Tunarr integration from supported functionality. Safely migrate existing installations without deleting source media or unrelated household data. Audit web and Android TV navigation, API authorization, session enforcement, tests and documentation for obsolete live-TV references. Verify on-demand playback, hard deadlines and session endings remain intact. Existing milestone numbering is retained for traceability. This is a removal task, not a new live-TV implementation.

## M7B Educational Arcade acceptance

Treat the arcade as a bonus, not a release dependency. Budget at most one working week (40 engineering hours, excluding independent content acquisition). Prioritize multiple-choice Quiz Show and True/False, then Picture Detective, Memory Match and picture/name/concept matching if time and approved assets permit. Classroom quizzes reuse the same quiz engine with teacher-controlled pacing. Reuse existing web and TV components; no separate game engine, elaborate animation, network multiplayer, progression systems or explorable worlds. Ship only tested formats; if the time budget expires or verified content is unavailable, disable unfinished formats and proceed to M8. Use prevalidated questions or deterministic templates, verified answers and distractors, age/subject metadata and licensed assets; no runtime AI dependency. Audit eligible content rather than promising question counts.

Post-v1.0 optional backlog (not release gates): odd one out, clue identification, fact explorer, bigger/smaller, higher/lower, timeline, geography, habitats, classification, fill in the blank, word scramble, image puzzles, fact-based arithmetic, larger knowledge adventures and expanded classroom competition. Reassess demand after release.

Parents and teachers can disable the arcade entirely or individual formats, choose age/grade, subjects, difficulty, sound and accessibility settings. Provide remote-friendly and touch-friendly controls, optional narration, untimed play, clear feedback and finite rounds. No ads, purchases, streak pressure, public leaderboards or automatic next rounds. Games respect hard session deadlines and never launch automatically after the end-of-session fact. Classroom quizzes support teacher-controlled presentation, pacing and optional local team scoring without requiring child accounts or public services. Test each shipped format for correctness, accessibility, offline operation, session limits and parental/teacher controls. Document any format deferred due to insufficient verified content. Include all shipped games in M10 user guides.

## M10 final documentation acceptance

Complete this milestone after feature development and release-candidate validation. Publish full, accurate guides for installation, setup, administration, parent and teacher workflows, Family and Classroom modes, content acquisition and screening, library organization, imports and exports, fun facts, accessibility, playback and session controls, backups, updates and troubleshooting, covering the features actually shipped.

Include an optional **Using AI to Help Manage Your Library** section with self-contained, easy copy-and-paste prompts for discovering and screening content, organizing age and subject metadata, identifying gaps, preparing classroom playlists, managing fun facts, preparing supported imports and troubleshooting. Include relevant documentation references and actual supported formats. Explain that AI suggestions require human review. This is documentation only: no built-in AI, API integration, subscription or AI infrastructure is required. FamilyTV remains fully functional without AI.

Verify the guides and prompts against the final release candidate. Publish the complete documentation before the v1.0 release. Maintain essential setup, safety and developer documentation throughout earlier milestones; defer the comprehensive end-user guides and prompt collection to this final milestone.
