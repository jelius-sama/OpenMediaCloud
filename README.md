# ClientToR2

A lightweight proxy that sits in front of a [Jellyfin](https://jellyfin.org) media server and redirects media requests directly to S3-compatible object storage (Cloudflare R2, AWS S3, etc.), bypassing the host server for media delivery entirely.

> [!NOTE]
> Though this application can sit in front of your jellyfin server it is recommended that you use a reverse proxy like caddy.
> Caddy can automatically manage TLS certificates, where as, our application does not and only focusses on handling the media redirection.

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
                             R2 / S3 Bucket ────────────────────┘
                           (media bytes served
                            directly to client,
                            EC2 not involved)
```

## Tradeoffs

### Tradeoffs that could improve

**No adaptive bitrate streaming (HLS) support**
Jellyfin uses ffmpeg to transcode media into HLS segments for clients with limited bandwidth or compatibility requirements. Since ClientToR2 redirects to the raw file on R2, transcoding is bypassed entirely. Clients that require HLS (such as the Jellyfin web client) may fall back to retrying or fail to play. The implemented solution is intercepting the `/Items/{id}/PlaybackInfo` response and nudging clients toward direct stream mode.
By intercepting the `/Items/{id}/PlaybackInfo` API to force Direct Stream mode, the proxy effectively bypasses the server's ability to transcode or decode media on behalf of the client.
Because R2/S3 serves the file directly, your client must natively support the original media format (codec/container). If the client cannot decode the source file, playback will fail. For the best experience, it is recommended to use widely supported video formats or a high-compatibility client (see the Codec Support section below for details).

**No quality selection**
Because media is served directly from R2 as the original file, users cannot select a lower quality stream when on a slow connection. Jellyfin's quality ladder relies on real-time transcoding through the server, which this architecture intentionally bypasses.

### TODOs

**Image and audio routes not yet intercepted**
Thumbnails, posters, and audio streams are still proxied through the VM. These are relatively small in size but could also be served directly from R2 (if you store the images there, usually jellyfin stores them in it's cache folder which may need to be mounted with rclone as well) with additional work following the same pattern as video.

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

# Codec Support

ClientToR2 redirects clients directly to the raw media file in R2/S3. There is no transcoding layer — the server does not decode or re-encode video on your behalf. What is stored in your bucket is exactly what the client receives.

This means playback success depends entirely on two things:

1. **Your client must natively support the codec of your media.** Native apps such as Infuse or SwiftFin tend to have broad codec support. Browser-based clients are limited to what the browser can decode, which typically means H264 and VP9 but not HEVC (especially 10-bit or 12-bit profiles) or AV1 on older devices.

2. **Your media should ideally be stored in a widely compatible format.** If broad client support matters to you, encode your media in H264 (High Profile, 8-bit) with AAC audio in an MP4 container. This combination plays on virtually every client without issues.

If a client cannot decode your media's codec, playback will fail silently or with a generic error. There is no fallback — this is an inherent constraint of the direct-redirect architecture.

---

# Storage Path Requirements

ClientToR2 resolves media by taking the file path that Jellyfin reports and using it directly as the S3/R2 object key. For this to work, **the directory structure visible to Jellyfin must match the directory structure in your bucket exactly**.

## Example of a working setup

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
> `/mnt/media` is [rclone](https://github.com/rclone/rclone) mount of your R2/S3 storage bucket.

Bucket structure:
```
your-media-bucket/
├── AMVs/
├── Anime/
└── HAnime/
```

Jellyfin sees the file at `/AMVs/akane edit __ capsize.mp4`. ClientToR2 strips the leading slash and uses `AMVs/akane edit __ capsize.mp4` as the S3 object key. The bucket must have the object at that exact key.

## Common mistake

If your Docker target is `/Anime` but your bucket folder is named `Anime Series`, the key will not resolve and playback will fail with a 404 from S3.

**The Docker target name and the bucket folder name must be identical.**

## Future improvement

A future release will introduce environment variables to explicitly map Jellyfin paths to S3 paths, removing this naming constraint. For now, the simplest solution is to ensure your bucket folder names match your Docker volume target names before ingesting media.

# Web Client Compatibility — Container Format and Codec

## TL;DR

If you want the Jellyfin web client to work correctly, store your media as **H.264 video, AAC audio, MP4 container, with faststart enabled**. MKV will not stream correctly in the browser regardless of codec or how the file is optimized, it will play BUT you will be using 2x the amount of data of your video file size (Eg. for 1.5GB size video you will end up using 3GB).

## Background

During development, extensive investigation was conducted into why the web client experienced a 3–5 minute delay before playback. The investigation went through several hypotheses before arriving at a definitive answer.

**What was ruled out:**
- The 307 redirect to S3 — the unbounded range request originated from the browser itself before any redirect occurred
- Container seek header placement — optimizing the MKV with `mkclean` to move seek headers to the front made no difference
- Jellyfin server coordination — Jellyfin simply serves whatever byte range it is asked for, it does not do anything special to help the browser locate the resume position
- Proxy architecture — the issue reproduces identically whether Jellyfin handles the request directly or the proxy redirects to S3

**What was confirmed:**

Testing with an MP4 file under the same conditions produced completely different behavior. The browser made three precise range requests:

1. A small read from the start of the file to locate and parse the `moov` atom
2. A small read from the end of the file to retrieve remaining index data
3. A direct read from the correct resume position for actual playback

The browser closed the first connection after receiving only ~34KB despite the server advertising the full file size. No pre-downloading occurred. Playback began from the correct position immediately.

The same browser with the MKV file downloaded the entire 1.50GB file before beginning playback.

**Root cause:**

The browser's MP4 parser is mature and performs surgical byte range reads to navigate the container structure with minimal data transfer. The browser does not have an equivalent MKV implementation and falls back to a linear download of the entire file instead.

This is a browser limitation and cannot be addressed at the proxy level.

## Recommendations

**For web client users:**

Store media as MP4 with H.264 video and AAC audio. When encoding or remuxing, using faststart flag would not make much difference as the browser already know how to properly parse mp4 container as such as long as you make sure that your video is a valid mp4 then the browser should handle the rest.

**For native client users (SwiftFin, Jellyfin Media Player):**

Native clients use platform media frameworks that handle MKV seeking correctly. Both MP4 and MKV work without issues and resume from the correct position within seconds. No specific encoding requirement applies.
