# FamilyTV handoff — 2026-11-23

M0 implementation underway on main. Go parent API, first-run setup, PBKDF2 password hashing, login lockout, HttpOnly SameSite cookies, same-origin POST checks, persisted local state, protected admin endpoint, React setup/login/dashboard, Docker multi-stage build, and CI source files are committed.

**Verification:** The locally developed M0 archive passed Go tests, race detector and go vet. The GitHub Go implementation was independently entered through the connector and is not yet verified by CI; do not conflate the two. Frontend dependencies could not be installed offline. Docker unavailable in the sandbox. No public release and no child use yet.

**Next:** verify GitHub Actions; align GitHub and local source, introduce PostgreSQL, secure external deployment, backup/export/delete, profile schema and accessible dashboard tests. Do not enter finalization early.

**Decisions pending:** final branding and human-made mascot, neither blocks development.
