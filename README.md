# ClientToR2

A lightweight proxy that sits in front of a [Jellyfin](https://jellyfin.org) media server and redirects media requests directly to S3-compatible object storage (Cloudflare R2, AWS S3, etc.), bypassing the host server for media delivery entirely.

> [!NOTE]
> Though this application can sit in front of your jellyfin server it is recommended that you use a reverse proxy like caddy.
> Caddy can automatically manages TLS certificates where as our application does not and only focusses on handling the media redirection.

## Why does this exist?

When you run Jellyfin on a cloud VM (such as AWS EC2) and store media on object storage (such as Cloudflare R2), the naive setup routes all traffic through your VM:

```
Client → EC2 (Jellyfin) → R2 → EC2 (rclone) → Client
```

This means every byte of video your users or you watch passes through your VM twice — once to fetch it from R2, and once to send it to the client. Moreover, for AWS EC2 users, AWS charges for outbound data transfer from EC2, and those costs add up quickly for a media server.

Cloudflare R2 has no egress fees, but only if the client fetches directly from R2. ClientToR2 makes that possible without replacing Jellyfin.

## How it works

ClientToR2 proxies all requests to Jellyfin as normal, except for media stream requests. When a media request is detected:

1. The item ID is extracted from the request URL.
2. Jellyfin's API is queried to resolve the item ID to a file path.
3. The file path is used to construct an S3 object key.
4. A presigned URL is generated pointing directly to the object in R2/S3.
5. The client is redirected (HTTP 307) to that presigned URL.

From that point, the client fetches media bytes directly from R2. The VM handles only the tiny redirect response.

```
Client → EC2 (ClientToR2) → 307 redirect → R2/S3 → Client
```

## Simplified Architecture Diagram

```

           ┌────────────────────────────────────────────────────┐
           │            ┌────────────────────────────────┐      │
           ▼            │           EC2 Instance         │      │
R2/S3 Server Response   │                                │      │
   Client Request ─────►│    ClientToR2 (this project)   │      │
Jellyfin server Response│            │                   │      │
        ▲               │    ┌───────┴────────┐          │      │
        │               │    │ Media request? │          │      │
        │               │    └───────┬────────┘          │      │
        │               │            │                   │      │
        │               │    YES     │    NO             │      │
        │               │    ┌───────┴──┐  ┌──────────┐  │      │
        │               │    │Presign   │  │ Jellyfin │  │      │
        │               │    │  URL     │  │  :8096   │  │      │
        │               │    └─────┬────┘  └──────┬───┘  │      │
        └───────────────┼──────────┼──────────────┘      │      │
                        └──────────┼─────────────────────┘      │
                                   │ 307 Redirect               │
                                   ▼                            │
                             R2 / S3 Bucket                     │
                           (media bytes served                  │
                            directly to client,                 │
                            EC2 not involved) ──────────────────┘
```

## Tradeoffs

### Tradeoffs that could improve

**No adaptive bitrate streaming (HLS) support**
Jellyfin uses ffmpeg to transcode media into HLS segments for clients with limited bandwidth or compatibility requirements. Since ClientToR2 redirects to the raw file on R2, transcoding is bypassed entirely. Clients that require HLS (such as the Jellyfin web client) may fall back to retrying or fail to play. A potential solution is intercepting the `/Items/{id}/PlaybackInfo` response and nudging clients toward direct stream mode, but this is not yet implemented.

**No quality selection**
Because media is served directly from R2 as the original file, users cannot select a lower quality stream when on a slow connection. Jellyfin's quality ladder relies on real-time transcoding through the server, which this architecture intentionally bypasses.

### TODOs

**Image and audio routes not yet intercepted**
Thumbnails, posters, and audio streams are still proxied through the VM. These are relatively small in size but could also be served directly from R2 with additional work following the same pattern as video.

### Tradeoffs that are architectural constraints

**Presigned URLs are visible to the client**
When the client is redirected to R2, the presigned URL is visible in the browser or app network logs. Presigned URLs are time-limited (default 1 hour) and scoped to a single object, so the risk is low, but they cannot be fully hidden by design — the client must have a URL it can fetch from directly.

**EC2 to R2 FUSE mount latency**
Jellyfin reads media metadata and generates thumbnails through the rclone FUSE mount. This means Jellyfin still fetches data from R2 for non-stream operations, which has higher latency than a local disk. This is inherent to using object storage as a filesystem and cannot be solved without a different storage architecture.

## Environment Variables

| Variable | Description |
|---|---|
| `JELLYFIN_HOST` | Full URL of your local Jellyfin server, e.g. `http://localhost:8096` |
| `JELLYFIN_API_KEY` | Jellyfin API key created under Admin → API Keys |
| `JELLYFIN_USER_ID` | Jellyfin user ID used to scope item lookups |
| `ACCESS_KEY_ID` | S3 / R2 access key ID |
| `SECRET_ACCESS_KEY` | S3 / R2 secret access key |
| `ACCOUNT_ID` | Cloudflare account ID |
| `AWS_REGION` | AWS region (S3 only, e.g. `us-east-1`). R2 uses `auto` |
| `S3_BUCKET` | Name of the S3 / R2 bucket storing your media |

## Switching between R2 and S3

The codebase targets R2 by default. To use AWS S3 temporarily, update `NewS3Client` in `internal/s3`:

- Remove the custom `BaseEndpoint` and `UsePathStyle` options
- Change the region from `auto` to your AWS region via `AWS_REGION`

Revert both changes when switching back to R2.
