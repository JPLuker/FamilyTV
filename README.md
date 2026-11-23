# FamilyTV (working name)

Self-hosted parent-controlled children's video platform. **Pre-release development scaffold. Not ready for use by children or public internet exposure.**

## Run
```sh
docker compose up --build
```
Open http://127.0.0.1:8080 for first-run parent setup. For development, run `cd backend && go run ./cmd/server` and `cd web && npm install && npm run dev`.

The parent dashboard and API share an origin in the Docker build. State is persisted in a private Docker volume. PostgreSQL, external TLS/reverse proxy guidance, backup/export/deletion, and full media handling remain unfinished.

## Network connections
| Host | When | Data |
|---|---|---|
| Self-hosted FamilyTV server | When using the dashboard | Parent authentication and application requests |
| npm, Go module and Docker registries | During explicit installation/build | Dependency requests |

No runtime analytics, tracking, advertising or external fonts. See ROADMAP.md and HANDOFF.md.
