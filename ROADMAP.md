# FamilyTV roadmap

**M1 media library, M2 profiles and M3 approvals are development-complete.** Automated acceptance uses synthetic media and profiles. Household and physical-device sign-off are reserved for testing. M4 child viewing is next in sequence. Working name and mascot pending Joseph's design decisions.

- **M0 Foundation:** Go API, parent authentication, web dashboard, PostgreSQL, Docker, CI, security and backups.
- **M1 Media library:** local import, FFprobe, playback, metadata and captions.
- **M2 Profiles and accessibility:** configurable reading/navigation level, assistive input and sensory settings.
- **M3 Approvals and Library Health:** parent-only approval, metadata review, accessibility and educational coverage.
- **M4 Child viewer:** approved-content-only discovery, TV focus, no on-demand autoplay.
- **M5 Sessions:** parent stop, after-video, duration and hard finish-by time; accessible ending.
- **M6 Live TV:** channels, EPG, optional Tunarr integration.
- **M7 Android TV:** remote-first app and pairing.
- **M7A Fun facts and Classroom Mode:** shared verified fact and authentic-image library, session endings, standalone fact presentation, teacher controls, grade/subject filters and lesson playlists.
- **M7B Educational Arcade:** optional fact-driven learning games, reusable local game engines and teacher-hosted classroom quizzes; complete before validation.
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

## M7B Educational Arcade acceptance

Build a reusable, offline-first educational arcade inspired by classic educational games and classroom quiz formats, without copying proprietary branding or assets. Target 20 formats, subject to verified data availability and accessibility: quiz show, true/false, picture detective, picture/name/fact matching, memory match, odd one out, clue-based identification, fact explorer, bigger/smaller, higher/lower, timeline ordering, geography explorer, habitat matching, classification sorting, fill in the blank, word scramble, image puzzle, fact-based arithmetic, knowledge adventure, and teacher-hosted classroom/team challenge. Start with quiz, picture identification, matching and memory; expand through shared engines rather than duplicating implementations. Audit the actual fact corpus and report eligible unique questions and activities by format, subject and age; do not promise a fixed count before inspecting it. Add structured entities, aliases, verified numeric values and units, dates, locations, taxonomy, habitat, approved questions, answers, distractors and explanations as applicable. Never infer unsupported answers from fact text; validate ambiguous questions and distractors, preserve provenance and image licensing, and skip ineligible records. Use prevalidated banks or deterministic templates; no runtime AI dependency.

Parents and teachers can disable the arcade entirely or individual formats, choose age/grade, subjects, difficulty, sound and accessibility settings. Provide remote-friendly and touch-friendly controls, optional narration, untimed play, clear feedback and finite rounds. No ads, purchases, streak pressure, public leaderboards or automatic next rounds. Games respect hard session deadlines and never launch automatically after the end-of-session fact. Classroom quizzes support teacher-controlled presentation, pacing and optional local team scoring without requiring child accounts or public services. Test each shipped format for correctness, accessibility, offline operation, session limits and parental/teacher controls. Document any format deferred due to insufficient verified content. Include all shipped games in M10 user guides.

## M10 final documentation acceptance

Complete this milestone after feature development and release-candidate validation. Publish full, accurate guides for installation, setup, administration, parent and teacher workflows, Family and Classroom modes, content acquisition and screening, library organization, imports and exports, fun facts, accessibility, playback and session controls, backups, updates and troubleshooting, covering the features actually shipped.

Include an optional **Using AI to Help Manage Your Library** section with self-contained, easy copy-and-paste prompts for discovering and screening content, organizing age and subject metadata, identifying gaps, preparing classroom playlists, managing fun facts, preparing supported imports and troubleshooting. Include relevant documentation references and actual supported formats. Explain that AI suggestions require human review. This is documentation only: no built-in AI, API integration, subscription or AI infrastructure is required. FamilyTV remains fully functional without AI.

Verify the guides and prompts against the final release candidate. Publish the complete documentation before the v1.0 release. Maintain essential setup, safety and developer documentation throughout earlier milestones; defer the comprehensive end-user guides and prompt collection to this final milestone.
