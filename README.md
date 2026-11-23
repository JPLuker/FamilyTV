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
| api.tvmaze.com | Parent selects Send lookup to TVmaze | Typed series query |
| www.youtube.com | Parent selects Send lookup to YouTube | Typed video ID |
| npm, Go module and Docker registries | During explicit installation/build | Dependency requests |

TVmaze lookup uses its public API, licensed under CC BY-SA. Selected matches retain their original listing link. See [TVmaze API terms](https://www.tvmaze.com/api).

No runtime analytics, tracking, advertising or external fonts. See ROADMAP.md and HANDOFF.md.

## Local media

Put an MP4 or WebM file in the project's media directory, or set FAMILYTV_MEDIA_ROOT to an existing directory before starting Compose. The host directory is mounted read-only. Enter the path relative to that folder in the parent dashboard. FamilyTV reads the video duration with FFprobe. A WebVTT file with the same base name supplies captions, for example Nature/Forest.vtt alongside Nature/Forest.mp4. Preview uses the browser's supported codecs. Back up the source media separately from the household database.

## Viewing sessions

Open Session next to a profile, choose the ending rule and start the session. Open the one-time viewer link on the child's browser within ten minutes. Use a server address that is reachable from that device. The default loopback deployment is for use on the same machine. Parent and viewer credentials are separate. A parent can stop a session immediately or at the end of the current video.

Finish by uses the time zone shown in the parent's browser. FamilyTV stores the absolute deadline and filters the catalog by remaining time, approval and required accessibility settings. The child starts each on-demand video. A lost connection pauses viewing until the service is available.

## Live channels

Parents create a channel by appending imported videos in playback order. The lineup starts at creation and repeats on a shared clock. The child guide shows the current and next eligible program. Joining live starts at the current program offset. Approvals, accessibility requirements and finish-by limits apply to the guide and stream. Continuous playback starts only after choosing a live channel.

## Android TV

The Android TV debug APK is attached to the android-tv CI job. Install it on a TV for testing, enter the HTTPS origin of your FamilyTV server, then the pairing code shown in the parent's session controls. Back opens pairing again. A new code replaces an unused one. Redeeming it moves the session to the new viewer. The TV stores the server address and uses the viewer cookie for the session.

### Tunarr guide import

In Tunarr, save the XMLTV guide from /api/xmltv.xml. In FamilyTV, open Live TV channels and select Import a Tunarr guide. Choose a station, review the local video matches and create the channel. The importer reads the file on the parent device. It imports up to 200 entries in broadcast order, skips entries the parent leaves unmatched and uses FamilyTV's own schedule and approvals. It makes no connection to Tunarr during playback.

### Scanning and matching

Scan the read-only folder from the dashboard and select the files to import. FamilyTV recognizes S01E02-style names, adjacent episode/movie NFO files and local video .info.json records. Creator, playlist and episode organization work offline. A sampled fingerprint identifies possible copies, then a complete byte digest confirms duplicates. Catalog details lets a parent correct a match or restore the originally imported metadata. Episode gaps describe holes up to the highest indexed episode.
