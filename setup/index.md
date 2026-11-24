---
layout: default
title: FamilyTV setup guide
---

[FamilyTV home](https://jpluker.github.io/FamilyTV/)

# Set up FamilyTV

FamilyTV runs on your server. You manage it from a browser on your phone or computer, and your child watches in a browser or the Android TV app. GitHub Pages hosts this guide. Your dashboard, videos and profiles stay on your server.

[Install on your server](#install-on-your-server) · [Share a Plex folder](#share-a-plex-folder) · [Approve a video](#approve-a-video) · [Connect your phone and TV](#connect-your-phone-and-tv) · [Test a session](#test-a-session)

## Install on your server

You'll need Git, Docker with the Compose plugin, and a video you can play locally. One short captioned episode is enough for the first test. Run these commands on the server:

```sh
git clone https://github.com/JPLuker/FamilyTV.git
cd FamilyTV
docker compose version
```

If you've already cloned FamilyTV, open that folder. Run the remaining Compose commands there.

## Share a Plex folder

Use the folder path on the server. If Plex maps `/mnt/media/TV:/tv`, FamilyTV needs `/mnt/media/TV`. To look up the path, use this command with jq and your Plex container's name. The left side of each line is the host path.

```sh
docker inspect plex | jq -r '.[0].Mounts[] | "\(.Source) -> \(.Destination)"'
```

Open or create `.env` beside `compose.yaml`. Add this line, replacing the example with your folder. Keep any settings already in the file.

```dotenv
FAMILYTV_MEDIA_ROOT="/mnt/media/TV"
```

FamilyTV reads this folder through a read-only mount. Importing creates a catalog entry. Plex keeps using the same file. If you'd rather try a single episode first, point FamilyTV at a smaller folder containing the video and its captions. For example, put `Forest.vtt` beside `Forest.mp4`.

### If another app uses port 8080

The default address is `127.0.0.1:8080`. If qBittorrent or another service already uses that port, choose a free one. With Compose 2.24.4 or newer, create or edit `compose.override.yaml` beside `compose.yaml` and set the API port:

```yaml
services:
  api:
    ports: !override
      - "127.0.0.1:18080:8080"
```

Use your chosen host port in the dashboard URL, SSH tunnel and reverse proxy below. The examples use the default port.

### Start FamilyTV

```sh
docker compose up -d --build
docker compose ps
curl -fsS http://127.0.0.1:8080/api/health
```

The health response should include `"status":"ok"`. Compose runs the FamilyTV web app and API together, plus a PostgreSQL container for household data.

## Approve a video

On the server, open `http://127.0.0.1:8080`. If the server has no desktop, run the following on your laptop. Replace `user@server` with your SSH login, keep the connection open, then visit `http://127.0.0.1:18081` on the laptop.

```sh
ssh -L 18081:127.0.0.1:8080 user@server
```

1. Create the parent account and sign in. Create a child profile.
2. Open **Scan the media folder**, then **Scan files**. Check one episode and choose **Import selected files**.
3. Choose **Preview** to check the file. Open **Review**, check the child under **Approved for**, then choose **Save review**. If the profile requires captions or described audio, verify those settings in the review too.

You can now test in a browser. Open **Session** beside the child's name, choose an ending rule and start it. Paste the one-time viewer link into a separate browser profile or private window on the same computer. Opening it in your parent browser signs that browser out.

## Connect your phone and TV

Complete parent setup first. The default address only works from the server. For a TV, use an HTTPS address on your home network that both the phone and TV can reach, with a certificate both devices trust.

If you already run an HTTPS reverse proxy on the server host, give FamilyTV its own hostname and point it at `127.0.0.1:8080`. Preserve the Host header. A proxy running in a separate container has its own loopback address, so that target only works for a proxy using the host network. Configure your household DNS and certificate for the hostname you choose. [Example Caddy configuration](https://github.com/JPLuker/FamilyTV/blob/main/SECURITY.md#https-on-the-household-network).

Add `FAMILYTV_SECURE_COOKIES=true` to the same `.env` file and run:

```sh
docker compose up -d api
```

Open your HTTPS address on your phone and sign in. Keep this development build on your household network.

1. Install the APK from the Android TV test issue. Download the ZIP, extract it and sideload the APK onto your Android TV. [APK and device checklist](https://github.com/JPLuker/FamilyTV/issues/1).
2. On the TV, enter your FamilyTV HTTPS address. Choose **Pair using a phone** and scan the QR code with your phone.
3. On the phone, open **Session** for the child and start a session if needed. Match the eight characters shown on the TV, then confirm the profile. Choose **Open paired viewer** on the TV.
4. To pair with the remote instead, open the child's session controls on your phone and enter the 13-character pairing code on the TV. A new code expires after ten minutes. Redeeming it moves that session to the new viewer.

## Test a session

Use the remote to choose the approved episode and select **Play on TV**. Check the audio and captions. Use Back to return to the episode, then to the list. Start playback again and choose **End now** on your phone. The TV should return to the session ending.

For the cutoff test, start a fresh session and choose **Finish by a clock time**. Leave enough time for the whole clip or it won't appear in the list. Start the clip and pause before the deadline. Check that the TV locks when time runs out.

## If setup gets stuck

| What you see | What to check |
| --- | --- |
| The server won't start | Run `docker compose logs --tail=50 api db`. If the port is in use, change the host port above. |
| The scan finds no files | Run `docker compose exec api ls -la /media`. Check the host path in `.env`, the container's read access and the file type. FamilyTV accepts MP4, MKV and WebM. Run `docker compose up -d api` after changing the path. |
| You can preview an episode but the child can't see it | Check **Approved for**, the profile's accessibility requirements and how much session time is left. |
| The dashboard works on the server but the TV can't connect | Open the same HTTPS address on your phone over Wi-Fi. Check household DNS, the certificate and the proxy's upstream port. |
| The pairing code expired | Open the child's active session and choose **Get a new pairing code**. |

PostgreSQL stores your child profiles, approvals and session settings. Keep a separate backup of the video files. [Back up or restore the household database](https://github.com/JPLuker/FamilyTV/blob/main/docs/BACKUPS.md).
