# Autonomous development protocol

Purpose: maximize completed, verified work between Joseph's interventions, without expanding approved scope or delaying v1.0.

## Loop
1. Read open issues, ROADMAP.md, DECISIONS.md, NO_LIST.md and HANDOFF.md.
2. Work on the earliest open milestone, starting with M1. Pick its highest-priority unverified acceptance criterion or a dependency needed to satisfy it.
3. Implement it, add tests, run relevant checks, update documentation.
4. Commit and push to main only after verification. Confirm the commit landed and check CI.
5. Record completed work and remaining failures in HANDOFF.md and issues.
6. Repeat from step 1 without requesting approval for in-scope work.

## Decision boundaries
Agent owns: implementation details, dependencies that meet security/privacy rules, refactoring with tests, bugs, test infrastructure, technical docs, deployment scripts, accessible functional UI behavior.
Joseph owns: final brand/name/mascot, human-made artwork, optional visual style/theme, fundamental product changes, irreversible external actions, acceptance on actual devices.

While a criterion is blocked, continue other work in the active milestone. Advance in numerical order after its acceptance criteria pass and the evidence is recorded. If every remaining criterion in that milestone is blocked, report the specific dependency before moving ahead. Record a single concise question in a decision issue and proceed elsewhere.

## Release boundaries
Scope freeze at release candidate. Fix release-blocking security, data loss, broken core journeys and critical accessibility issues. Move non-blocking polish and new ideas to post-v1.0. A milestone completes only when its acceptance tests pass; a passing CI alone is not proof of usability.

## Checkpoint requirements
Every checkpoint: current commit SHA, files changed, exact tests and results, unverified areas, next actionable task. Do not claim success on unrun tests. Never create fake evidence.

## Limits
An ordinary chat turn cannot execute indefinitely after the reply ends. Use a continuing Work session for long-running autonomous execution if available; recurring automations require separate opt-in and may not provide an uninterrupted coding environment.

## Reuse before building

Follow the [shared source-code reuse standard](https://github.com/JPLuker/InternalDesignGuidelines#reuse-existing-code-before-building-from-scratch). For media work, inspect Jellyfin or another mature media project's implementation and tests before writing a new parser, matcher or playback routine. Adapt or port suitable code directly into FamilyTV when the license and maintenance cost fit. Keep local playback standalone and verify that the reused code respects FamilyTV's profile approvals, revocation and session deadlines.
