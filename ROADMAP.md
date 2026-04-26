## Roadmap

### 1 — CloudFront transcoding (HLS support with caveats)

For CloudFront users, there is a potential path to bringing back HLS support and adaptive quality selection. The idea is to use ffmpeg on the EC2 instance to transcode media on demand, store the resulting HLS segments in S3, and serve them through CloudFront.

This works without incurring egress costs because all data movement — downloading the source from S3, transcoding, and uploading HLS segments back to S3 — happens within the AWS network when your EC2 instance and S3 bucket are in the same region. Only the final delivery from CloudFront to the client counts against your 1TB free tier.

**Why this only works for S3 + CloudFront, not R2:**
After transcoding, the HLS segments need to be written back to object storage. EC2 to R2 transfer travels over the public internet, which AWS charges egress for. EC2 to S3 within the same region is free. This approach is therefore exclusive to users already on the AWS stack.

**Implementation considerations:**
- Transcode on demand when a request arrives, or pre-transcode on upload
- Tag transcoded artifacts in a KV store to avoid redundant work on repeated requests
- Store segments in a dedicated `transcoded/` prefix in the bucket, keyed by item ID and quality level
- Serve segments through CloudFront using the same signed URL mechanism already in place
- Handle cleanup of old or unused transcoded artifacts to avoid unbounded storage growth
- Real-time transcoding of 4K source material is slow — downloading the source, transcoding, uploading segments, and serving all in one pipeline introduces significant latency before playback begins. Pre-transcoding on upload is the more practical path.
- Real-time transcoding may be painfully slow, it would be good if we can have two options either decode at upload time or on demand, if we have to decode at upload time, we may need to keep the decoded asset for a long period of time which could incur more storage cost, on the otherhand users with beafy instance can do real time decoding/transcoding.

### 2 — Image serving through CloudFront

Applying the same strategy as roadmap item 1 to images — uploading Jellyfin-generated thumbnails and posters to S3 and serving them through CloudFront. Since Jellyfin stores images in its cache directory rather than in your media bucket, this requires either mounting the cache directory with rclone or implementing a write-through cache that uploads images to S3 the first time they are requested and tracks them in a KV store for subsequent redirects.

Images are small so the egress savings are minimal, but this would eliminate the last category of requests that still pass through the EC2 instance.

Note that the 1TB CloudFront free tier works out to roughly 33GB per day — images are unlikely to make a meaningful dent in that budget.

### 3 — External subtitle and supporting asset delivery

Handle media assets that live alongside the video file but outside the container — things like `.srt`, `.ass`, or forced subtitle files (e.g. `EP1.forced.srt` next to `EP1.mp4`). These follow the same pattern as video: intercept the request, resolve the file path, generate a presigned URL, and redirect. The main work is identifying which Jellyfin API routes serve these assets and mapping them correctly to their S3 object keys.

### 4 — Path mapping via environment variables

Allow users to define explicit mappings between Jellyfin-visible paths and S3 object key prefixes through environment variables, removing the current requirement that Docker volume target names match bucket folder names exactly. This would make the setup more flexible and less error-prone for users whose existing bucket structure does not align with their Docker volume naming.

## Immich
Immich has official API documentation at `https://api.immich.app/` — that's the best place to browse everything interactively.

---

Here are all the endpoints that deliver heavy media files specifically:

**Asset viewing/streaming**
- `GET /api/assets/{id}/original` — downloads the original full-resolution photo or video file
- `GET /api/assets/{id}/thumbnail` — retrieves the thumbnail (lighter but still media)
- `GET /api/assets/{id}/video/playback` — streams video for playback

**Download**
- `GET /api/assets/{id}/original` — same endpoint, used for direct download
- `POST /api/download/archive` — downloads multiple assets as a `.zip` stream, accepts a list of asset UUIDs or an album ID

**Upload (inbound heavy media)**
- `POST /api/assets` — uploads a new asset as `multipart/form-data` with the binary file

**Profile/person images**
- `GET /api/people/{id}/thumbnail` — face/person thumbnail

**Albums**
- No direct media delivery, but `POST /api/download/archive` accepts `albumId` to download an entire album as zip

---

The full OpenAPI spec has over 400 endpoints across 30+ functional categories — for anything beyond media delivery the official docs at `api.immich.app` are the most complete reference.
