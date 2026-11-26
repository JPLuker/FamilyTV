# Security and privacy

FamilyTV is a development build for a household-controlled installation. Keep the database and media mounts private. Use the repository's private vulnerability reporting option if enabled, or contact the maintainer through their GitHub profile before publishing details involving credentials or household data.

| Boundary | Controls and outstanding review |
| --- | --- |
| First parent setup | Compose publishes the API on loopback. Create the parent account there before allowing other devices to connect. Parent sessions use password verification, expiring HttpOnly cookies and per-network-peer login limits. Parents may also require a 6–12 digit PIN with the password. Changing the PIN invalidates other parent sessions. |
| Shared viewer devices | A parent creates a short-lived single-use code. The viewer gets a distinct cookie scoped to the watch API. Redeeming a link invalidates the parent session in that browser. Profile device codes grant a watch-only credential for up to one year. Disconnecting devices invalidates the profile’s persistent credentials and pending device code. Media access additionally requires an active viewing session and approval. Legacy session codes continue to transfer only the temporary viewing credential. |
| Playback and discovery | The API checks profile, approval, local file identity, accessibility and deadline. Existing streams periodically recheck current state, including database-backed revocation. A viewer may already have received media bytes before a stop. FamilyTV provides household controls, not DRM. |
| Local media | A read-only mount confines file access. Import and preview invoke FFmpeg tools with a supported container allowlist and a timeout. Only import files the household trusts. Keep FFmpeg, the operating system and container images updated. |
| Stored household data | Household state includes preferences, approvals and active credentials. PostgreSQL uses a private Unix socket in Compose. A verified profile deletion removes the profile, session and approval assignments from live storage. Backups and the original migration source have separate retention. |
| Network privacy | Direct playback and recommendations run locally. No analytics, ad requests or public account service. Browser tests and builds download dependencies. Installing Android TV requires Android/Gradle repositories. |
| Release review | Independent security review, child privacy applicability review and real-device accessibility testing remain open. Automated tests exercise specific paths. |

## HTTPS on the household network

Complete setup from loopback first. Run a TLS-terminating reverse proxy on the host. Preserve the Host header and keep the API bound to loopback. For example, a Caddyfile for a hostname you control can contain the following.

```caddyfile
familytv.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

Set FAMILYTV_SECURE_COOKIES=true for the API service and recreate it before using the HTTPS address. The phone and TV must trust the certificate and resolve the hostname. The TV client requires HTTPS. A reverse proxy shares one login-rate-limit bucket across its viewers. Do not publish PostgreSQL or enable unauthenticated remote setup. See [Caddy reverse proxy documentation](https://caddyserver.com/docs/quick-starts/reverse-proxy).

Parent-only review history retains the latest 200 actions, including video titles and profile identifiers. It excludes watch history and credentials. Child names shown alongside actions come from current profiles. The household backup includes the review history. Coverage reads local file availability when the parent opens the approval queue.
