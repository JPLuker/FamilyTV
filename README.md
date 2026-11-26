# FamilyTV (working name)

Self-hosted parent-controlled children's video platform. **Pre-release development scaffold. Not ready for use by children or public internet exposure.**

[FamilyTV](https://jpluker.github.io/FamilyTV/) · [Setup guide](https://jpluker.github.io/FamilyTV/setup/) · [Read the guide on GitHub](docs/guide/setup/index.md)

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

Put an MP4, MKV or WebM file in the project's media directory, or set FAMILYTV_MEDIA_ROOT to an existing directory before starting Compose. The host directory is mounted read-only. Enter the path relative to that folder in the parent dashboard. FamilyTV reads the video duration with FFprobe. A WebVTT file with the same base name supplies captions, for example Nature/Forest.vtt alongside Nature/Forest.mp4. Preview uses the browser's supported codecs. Back up the source media separately from the household database.

## Viewing sessions

Open Session next to a profile, choose the ending rule and start the session. Open the one-time viewer link on the child's browser within ten minutes. Use a server address that is reachable from that device. The default loopback deployment is for use on the same machine. Parent and viewer credentials are separate. A parent can stop a session immediately or at the end of the current video.

Finish by uses the time zone shown in the parent's browser. FamilyTV stores the absolute deadline and filters the catalog by remaining time, approval and required accessibility settings. The child starts each on-demand video. A lost connection pauses viewing until the service is available.

## Android phones, tablets and TV

The same APK opens from the Android app drawer or TV home screen. Phones and tablets support touch and rotation in pairing, browsing and Media3 playback. Enter the server address and a six-character code from the parent dashboard. TV also supports remote navigation and phone-assisted pairing. For LukerPlus testing on trusted Wi-Fi, the debug APK accepts `http://192.168.86.163:8095` when entered explicitly. HTTP exposes viewer traffic to the network. The release configuration requires HTTPS.

In **Household devices**, choose **Get pairing code**. You can link a device before creating a child profile. Enter the code on the device. For TV QR pairing, scan the TV’s QR code and use **Household devices** on the parent dashboard to confirm the TV identifier. A device can pair before the first session. Select the child in the screen’s **Viewing profile** menu, then start viewing with **Session**. Change or clear that assignment anytime. Ending a session locks paired screens, and the next session unlocks them. Each code lasts ten minutes. Device access lasts up to one year or until disconnected in the dashboard.

Under **Media library**, use **Scan now** to recursively import the mounted folder. The server also scans at startup and hourly, skips previously imported paths and duplicate files, and leaves new videos awaiting parent review. Use the approval queue to select and approve a reviewed batch for one child.


### Scanning and matching

Scan the read-only folder from the dashboard and select the files to import. FamilyTV recognizes S01E02-style names, adjacent episode/movie NFO files and local video .info.json records. Creator, playlist and episode organization work offline. A sampled fingerprint identifies possible copies, then a complete byte digest confirms duplicates. Catalog details lets a parent correct a match or restore the originally imported metadata. Episode gaps describe holes up to the highest indexed episode.

### Child viewing and playback

Each subject heads its own group of videos. Subject order is shuffled for each new session and stays stable while browsing. Open Options and choose Save for later to take a video out of today’s main choices. It gets priority the next session it fits and leaves the saved list when finished. Back to choices keeps the current position. Finish video clears the position. In a one-video or finish-after-current session, leaving the video ends the session. Surprise me selects from the currently displayed choices. On-demand playback waits for Play.

The viewer tries the source file first. On decoder failure the server prepares a seekable MP4 with FFmpeg, copying supported H.264 video and AAC audio where possible. If the client still cannot decode it, it can retry with both tracks encoded. Compatibility playback uses the default audio track and the same WebVTT sidecar. Preparation runs one job at a time with a 15-minute limit and a 512 MiB output limit. A large source can take time to prepare or exceed these limits. One anonymous temporary copy is cached for repeat requests. Viewer access is checked during preparation and delivery.

### Viewing windows and allowances

In a child profile, open Viewing windows and daily allowances. Windows follow the selected time zone and can cross midnight. Daily allowances count elapsed session minutes, including browsing and pauses. A category uses minutes while its video is selected. Limits apply to new sessions. A metered session ends at midnight or when a limit is reached; unused minutes do not carry forward. Moving to another screen keeps the session clock.

Choose an ending fact topic or enable Quiet ending for a message-only finish. Optional music and facts end within twelve seconds. A hard cutoff goes directly to the locked message.

### Upgrading from a release with live channels

Back up the household before updating. The server archives existing lineups in the household backup and ends tuned sessions. On-demand sessions and imported videos remain available. To roll back, restore the pre-upgrade backup with the corresponding server version.

### Android setup and playback

The app searches for FamilyTV at setup. The debug build checks the configured LukerPlus address automatically. Other installations can advertise their HTTPS origin using DNS-SD: on the LAN host with Avahi running, copy `deploy/familytv.service` to `/etc/avahi/services/` after editing the port and origin to match the address trusted by your devices. Manual entry is under **Advanced server address**.

Choose **Parent mode** to open the authenticated dashboard in the device browser. Set the parent PIN under **Parent sign-in settings**. Before generating a device code or authorizing a TV, optionally allow that screen to start a one-video session. Assign a viewing profile after linking. Each viewer-initiated start requires the parent PIN and respects the assigned profile's viewing windows and allowances. In parent mode the dashboard can start sessions for other profiles and their assigned devices.

Tap a tile or press OK to play. The tile’s menu holds Save for later. The player returns to choices when the video finishes. During import and scan, FamilyTV fills category and subject from available local sidecar categories, genres and recognized topic metadata. Review still controls approval.
