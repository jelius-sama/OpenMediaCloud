# OpenMediaCloud

A lightweight proxy that sits in front of a [Jellyfin](https://jellyfin.org) media server and redirects media requests directly to S3-compatible object storage (Cloudflare R2, AWS S3, etc.) or Cloudfront, bypassing the host server for media delivery entirely.

> [!NOTE]
> Though this application can sit directly in front of your Jellyfin server, it is recommended that you place a reverse proxy like [Caddy](https://caddyserver.com) in front of it. Caddy can automatically manage TLS certificates, whereas this application focuses solely on media redirection.

---

## Why does this exist?

When you run Jellyfin on a cloud VM (such as AWS EC2) and store media on object storage (such as Cloudflare R2), the naive setup routes all traffic through your VM:

```
Client → EC2 (Jellyfin) → R2 → EC2 (rclone) → Client
```

Every byte of video passes through your VM twice — once fetched from R2, once sent to the client. AWS charges for outbound EC2 data transfer, and those costs add up quickly for a media server.

Cloudflare R2 has no egress fees, but only if the client fetches directly from R2. OpenMediaCloud makes that possible without replacing Jellyfin.

---

## How it works

OpenMediaCloud proxies all requests to Jellyfin as normal, except for media requests. When a media request is detected:

1. The item ID is extracted from the request URL.
2. Jellyfin's API is queried to resolve the item ID to a file path.
3. The file path is used to construct an S3 object key.
4. A presigned URL is generated pointing directly to the object in R2/S3.
5. The client is redirected (HTTP 307) to that presigned URL.

From that point, the client fetches media bytes directly from R2/S3. The VM handles only the tiny redirect response.

```
Client → EC2 (OpenMediaCloud) → 307 redirect → R2/S3 → Client
```

### Architecture Diagram

```
           ┌────────────────────────────────────────────────────┐
           │            ┌────────────────────────────────┐      │
           ▼            │           EC2 Instance         │      │
R2/S3 Server Response   │                                │      │
   Client Request ─────►│        OpenMediaCloud          │      │
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
                             R2 / S3 Bucket ────────────────────┘
                           (media bytes served
                            directly to client,
                            EC2 not involved)
```

---

## What works

- **Video streaming** — intercepted and redirected to R2/S3 via presigned URL
- **Audio streaming** — intercepted and redirected to R2/S3 via presigned URL
- **Downloads** — intercepted and redirected to R2/S3 via presigned URL
- **Authentication** — client tokens are validated against Jellyfin before any media redirect is issued
- **HLS / Direct stream negotiation** — the `/Items/{id}/PlaybackInfo` response is patched to force direct stream mode, bypassing transcoding entirely
- **Everything else** — forwarded transparently to Jellyfin (metadata, images, search, user management, etc.)

---

## Tradeoffs

### Potential to improve

**No adaptive bitrate streaming**
By forcing direct stream mode via `PlaybackInfo` patching, transcoding is bypassed entirely. Users on slow connections cannot fall back to a lower quality stream. Jellyfin's quality ladder relies on real-time transcoding through the server, which this architecture intentionally skips.

**Image serving still goes through the VM**
Thumbnails and posters are forwarded to Jellyfin and served through your VM. Images are small so the egress cost is negligible, but a future implementation could cache them in R2 using a key-value store to track which images have already been uploaded, serving subsequent requests via presigned URL the same way video is handled.

### Architectural constraints

**Presigned URLs are visible to the client**
When the client is redirected to R2, the presigned URL is visible in browser and app network logs. Presigned URLs are time-limited (default 1 hour) and scoped to a single object so the practical risk is low, but they cannot be fully hidden — the client must have a URL it can fetch from directly.

**EC2 to R2 FUSE mount latency**
Jellyfin reads media metadata and generates thumbnails through the rclone FUSE mount. Non-stream operations still go through the mount, which has higher latency than local disk. This is inherent to using object storage as a filesystem and cannot be solved without a fundamentally different storage architecture.

> [!NOTE]
> rclone's `--vfs-cache-mode` option can partially mitigate this by caching frequently accessed files locally on the VM, reducing latency for repeated metadata operations.

---

## Storage Path Requirements

OpenMediaCloud resolves media by taking the file path Jellyfin reports and using it directly as the S3/R2 object key. For this to work, **the directory structure visible to Jellyfin inside Docker must match the directory structure in your bucket exactly**.

### Example of a working setup

Jellyfin Docker volume mounts:
```yaml
volumes:
  - source: /mnt/media/AMVs
    target: /AMVs
  - source: /mnt/media/Anime
    target: /Anime
  - source: /mnt/media/HAnime
    target: /HAnime
```

> [!NOTE]
> `/mnt/media` is an [rclone](https://github.com/rclone/rclone) mount of your R2/S3 storage bucket.

Bucket structure:
```
your-media-bucket/
├── AMVs/
├── Anime/
└── HAnime/
```

Jellyfin sees the file at `/AMVs/akane edit __ capsize.mp4`. OpenMediaCloud strips the leading slash and uses `AMVs/akane edit __ capsize.mp4` as the S3 object key. The bucket must have the object at that exact key.

### Common mistake

If your Docker target is `/Anime` but your bucket folder is named `Anime Series`, the object key will not resolve and playback will fail with a 404 from S3.

**The Docker volume target name and the bucket folder name must be identical.**

> [!NOTE]
> A future release will introduce environment variables to explicitly map Jellyfin paths to S3 paths, removing this naming constraint entirely.

---

## Codec and Container Support

OpenMediaCloud redirects clients directly to the raw media file in R2/S3. There is no transcoding layer — what is stored in your bucket is exactly what the client receives. Playback success depends entirely on whether the client can natively decode the source file.

### Web client (browser)

The Jellyfin web client has a known limitation with MKV files — the browser downloads the entire file before beginning playback regardless of file size, seek headers, or any proxy-level optimization. This was confirmed through extensive testing and is a browser MKV parser limitation that cannot be addressed at the proxy level.

**Use MP4 with H.264 video and AAC audio for the web client.** The browser's MP4 parser is mature and performs surgical byte range reads to locate the resume position with minimal data transfer — typically under 100KB before playback begins.

MKV will play on the web client but will consume approximately 2x the file size in data transfer per session (e.g. a 1.5GB episode costs ~3GB of data transferred) due to the full pre-download.

### Native clients (SwiftFin, Jellyfin Media Player, Infuse)

Native clients use platform media frameworks that handle both MP4 and MKV seeking correctly. They resume from the correct position within seconds regardless of container format. No specific encoding requirement applies.

### General recommendation

If broad client compatibility matters, encode media as H.264 (High Profile, 8-bit), AAC audio, MP4 container. This combination plays correctly on every tested client without issues.

---

## Environment Variables

| Variable | Description |
|---|---|
| `JELLYFIN_HOST` | Full URL of your local Jellyfin server, e.g. `http://localhost:8096` |
| `JELLYFIN_API_KEY` | Jellyfin API key created under Admin → API Keys |
| `JELLYFIN_USER_ID` | Jellyfin user ID used to scope item lookups via the Items API |
| `ACCESS_KEY_ID` | S3 / R2 access key ID |
| `SECRET_ACCESS_KEY` | S3 / R2 secret access key |
| `AWS_REGION` | Storage region. R2 uses `auto`. AWS S3 uses a region code e.g. `us-east-1`. |
| `BUCKET_NAME` | Name of the S3 / R2 bucket storing your media |
| `BASE_URL` | In case you are not using AWS S3 you want to set the base URL as per your provider. (default: unset; uses AWS S3 endpoint as base) |
| `CLOUDFRONT_ENDPOINT` | If set then uses Cloudfront instead of AWS S3. (default: unset; uses AWS S3) |
| `CLOUDFRONT_KEY_PAIR_ID` | Key ID of your Public Key when using Cloudfront signed URL. |
| `CLOUDFRONT_PRIVATE_KEY_PATH` | Absolute path to your Private Key associated with your Public key. |

---

## Storage Backend Options

OpenMediaCloud targets any S3-compatible storage backend. The recommended option is Cloudflare R2 due to its zero egress fees, which is the primary motivation for this project.

**Cloudflare R2** — zero egress fees, S3-compatible, recommended default. Set `BASE_URL` and use region `auto`.

**AWS S3 with CloudFront** — if you prefer AWS, pairing S3 with a CloudFront distribution increases your effective free egress from 100GB to 1TB per month.

**Other S3-compatible storage** — any S3-compatible provider (Backblaze B2, Wasabi, etc.) should work by pointing the custom `BASE_URL` at the provider's S3-compatible endpoint.
