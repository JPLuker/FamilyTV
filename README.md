# FamilyTV (working name)

Self-hosted parent-controlled children's video platform. **Pre-release development scaffold. Not ready for use by children or public internet exposure.**

## Run
```sh
docker compose up --build
```
Open http://127.0.0.1:8080 for first-run parent setup. For development, run `cd backend && go run ./cmd/server` and `cd web && npm install && npm run dev`.

The parent dashboard and API share an origin in the Docker build. Compose stores household data in PostgreSQL 17 on a private volume. The API connects through a local Unix socket. First startup imports the previous JSON file if present. Standalone Go development still supports a JSON file. See [backup and recovery](docs/BACKUPS.md).

## Network connections
| Host | When | Data |
|---|---|---|
| Self-hosted FamilyTV server | When using the dashboard | Parent authentication and application requests |
| npm, Go module and Docker registries | During explicit installation/build | Dependency requests |

No runtime analytics, tracking, advertising or external fonts. See ROADMAP.md and HANDOFF.md.
