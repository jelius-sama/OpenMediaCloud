# The Architecture Problem Nobody Warned Me About

When I started building OpenMediaCloud, the idea was simple: I was paying for AWS EC2 to run Jellyfin, storing media on Cloudflare R2 because the storage cost is cheaper and egress is free, but every byte of video was routing through my EC2 instance twice — once in from R2 via rclone, once out to the client. AWS bills for outbound EC2 data transfer. For a media server that streams gigabytes at a time, that adds up fast.

The fix seemed obvious. Intercept media requests, resolve the file path, generate a presigned URL pointing directly at R2, and redirect the client there. The EC2 instance handles only a tiny redirect response. The client fetches directly from R2. Zero egress cost.

That worked. It works well. But as soon as I started extending the same idea to Immich for photo serving, and then started thinking about Komga for manga, the simplicity started to crack — and what began as a lightweight proxy started revealing a much deeper architectural problem.

---

## Where the Simple Model Breaks

The presigned URL trick works under one assumption: the file the client wants already exists as a discrete object in storage. For video files and photos that is true. The client wants `/Anime/series/episode.mp4` — that object is in the bucket, generate a URL, redirect, done.

But several common operations do not work this way.

**Immich archive downloads.** When a user selects multiple photos and hits download, Immich assembles them into a ZIP on the fly. No ZIP exists in storage — it is created on demand from individual assets. You cannot redirect a client to an object that has never been written.

**Jellyfin transcoding.** When a client cannot play the source codec, Jellyfin transcodes in real time. The transcoded stream does not exist as a file anywhere. You cannot redirect to nothing.

**Komga page serving.** This one is the most fundamental. Komga stores manga as CBZ archives — ZIP files containing a sequence of images. Its page endpoint (`GET /api/v1/books/{bookId}/pages/{pageNumber}`) extracts the requested image from the archive and streams it as raw bytes. Every page request is an extraction operation. The individual page images do not exist anywhere as standalone files, and there is no alternate API path that serves them differently. This is not an edge case — it is how Komga works for every single page of every single book.

In all three cases, the server needs to do computational work before anything can be sent to the client. A redirect-only proxy has no answer for this.

---

## The Workarounds I Have Today

For Jellyfin transcoding, there is a partial mitigation. The proxy intercepts the playback info response and patches it to force direct stream mode, steering the client into playing the original file rather than requesting a transcode. This works well on native clients — SwiftFin, Jellyfin Media Player, Infuse — which have broad codec support. It does not work for web browser clients, which have limited format support and will fail to play content the browser cannot decode natively. For my own usage, this is acceptable. For a general audience it is a documented limitation.

For Immich archive downloads, I simply pass the request through to Immich and let it handle the work. Archive downloads are infrequent enough that the EC2 egress cost is manageable as a one-off. Not elegant, but not a real problem in practice either.

For Komga, there is no equivalent mitigation. There is no mode where Komga redirects a client to a file. Every page request goes through the server. On a busy reading session — flipping through a manga chapter — that is fifty to two hundred sequential requests, each one serving raw image bytes out of EC2. The egress from Komga alone could dwarf everything else combined.

---

## What This Problem Is Actually About

Sitting with this for a while, the underlying issue became clear: the architecture of the proxy and the architecture of the infrastructure are coupled in a way they should not be.

The current setup assumes that everything the client could want either already exists in storage or can be handled by the upstream server at acceptable cost. When neither of those is true — when you need compute, and compute produces output that must then be delivered cheaply — the proxy has no tools to deal with it.

The direction I have been thinking about is a different division of responsibilities between storage providers. Not one bucket for everything, but deliberate placement of data based on the cost profile of each service and how that data will be accessed.

---

## A Different Architecture

The shape I have been working toward uses three services in distinct roles:

**Cloudflare R2** as the primary media storage. Free egress when clients fetch directly from R2. Cheaper per-GB storage than S3 for bulk media. This is where Jellyfin media and Immich photos live — everything that already exists as a discrete file and can be served with a presigned redirect.

**AWS S3** as compute output storage and Komga's primary storage. S3 paired with an EC2 instance in the same region, connected via a VPC Gateway Endpoint so data movement between EC2 and S3 stays on AWS's internal network and does not incur transfer charges. This is where generated outputs land — transcoded videos, assembled archives, extracted Komga pages — written once by EC2, read repeatedly by clients via CloudFront.

**AWS CloudFront** as the delivery layer for everything in S3. CloudFront includes 1TB of free egress per month, which covers a substantial amount of media delivery before any cost is incurred. Data flows from S3 to CloudFront to the client, with EC2 never in the path for delivery.

**EC2** purely as a compute unit. It runs Jellyfin, Immich, and Komga. When computation is needed — extracting a Komga page, assembling a ZIP, producing a transcode — EC2 does the work and writes the result to S3. The result is cached there. Subsequent requests for the same output skip computation entirely and go directly to CloudFront.

The 100GB of free EC2 outbound transfer that AWS includes handles the small amount of traffic that must flow directly from EC2 — API responses, auth checks, anything the proxy cannot redirect. The actual media bytes move through CloudFront or directly from R2, not through EC2 at all.

The economics work because each piece is doing what it is cheapest at. R2 stores and delivers the large bulk of media at zero egress cost. S3 and CloudFront handle the compute outputs where the free tier covers most realistic usage. EC2 computes but never delivers — its expensive outbound bandwidth is not used for media bytes.

---

## What This Is Not

I want to be clear about the status of what I just described: it is a direction, not a shipped feature. I will set this up for my own infrastructure first, manually, to validate that the economics and the architecture actually work the way I expect. Experiments on paper do not always survive contact with real traffic and real billing.

If it works and the operational complexity is manageable, the longer-term possibility is making this architecture configurable from within OpenMediaCloud itself — where the proxy understands which storage backend handles which kind of data, and routes accordingly without manual configuration. Something closer to architecture-as-code, where you declare your infrastructure and the proxy optimizes around it.

But I have real reservations about that direction. This is a niche space — the overlap between people who self-host Jellyfin, store media on object storage, and care enough about egress costs to run a proxy in front of their setup is not large. Building and maintaining opinionated architecture management for that audience is a significant ongoing commitment. More fundamentally, I am not sure I am comfortable with the idea of software making infrastructure decisions on someone else's behalf, even open source software. That kind of control has to be earned through a long track record of correctness, and I do not think any project gets there quickly.

So the manual path will always be available. S3-compatibility is maintained throughout — you can substitute R2 or S3 with any S3-compatible provider (Backblaze B2, Wasabi, MinIO, whatever) at any layer. The architecture described here is a recommended configuration, not a requirement.

Whether the architecture-as-code vision happens at all depends on whether the manual setup proves out, whether the operational reality matches the theory, and honestly whether I still feel like building it after living with the manual version for a while. No promises.

What I can say is that the problem is real, the direction is coherent, and the first step — validating it manually on my own infrastructure — is the right one before any of the larger decisions get made.
