# Architectural Workarounds for Compute-Dependent Features

This document covers features that cannot be handled by a simple redirect to object storage because they require on-demand computation. It explains why certain features are not currently implemented, what the architectural constraints are, the workaround being planned, and every caveat you need to understand before attempting to use it.

---

## What features are affected?

Some requests cannot be served by redirecting the client to a pre-existing file in S3/R2. The file does not exist yet — it has to be created on demand from one or more source files. Examples:

- **Immich: Download Archive** — The client requests a ZIP archive containing multiple assets. No such ZIP exists in storage; it must be assembled at request time.
- **Jellyfin: Transcoding** — A client that cannot play the source codec/container requests a transcoded version. No transcoded file exists in storage; it must be produced by FFmpeg at request time.

Both of these require a compute unit — something that can pull source files, do work, and produce an output — which is fundamentally outside the scope of a redirect-only proxy.

---

## Why not just pass the request to the upstream server?

You could. Jellyfin and Immich both handle these natively. But doing so defeats the purpose of this project: the bytes would flow out of the server directly to the client, and AWS charges for that outbound EC2 data transfer. For a single archive download or one transcoded video that is not a problem, but at any real usage scale it adds up in exactly the way this project exists to avoid.

---

## Why not use Cloudflare Workers as the compute unit?

Jellyfin and Immich are not serverless applications. They run as persistent processes inside Docker containers and cannot be hosted on Cloudflare Workers. But more importantly, the compute task itself — zipping multiple files, transcoding a video with FFmpeg — is a heavy, CPU-bound operation that can run for seconds or minutes.

Cloudflare Workers bill on CPU time. A serverless function that runs FFmpeg for 30 seconds on a large video file is not a use case Workers are designed for, and the cost would be disproportionate. Serverless billing horror stories exist for exactly this kind of workload: a task that seems cheap per request scales catastrophically when usage increases. Workers are appropriate for lightweight, latency-sensitive tasks (routing, edge logic, small transformations), not media processing.

EC2 is the correct compute unit here. You are already paying for it by the hour regardless of what it does, so using its CPU for occasional transcoding or zip generation costs nothing incremental.

---

## The proposed workaround

The approach that avoids egress costs while still using EC2 for compute is:

1. EC2 receives the compute request (transcoding, archive generation, etc.)
2. EC2 pulls the necessary source assets from R2 — **ingress to EC2 is free**
3. EC2 does the computation (FFmpeg, zip, etc.)
4. EC2 writes the output file to an S3 bucket — **free if EC2 and S3 are in the same region, with prerequisites described below**
5. CloudFront serves the output file to the client — **1 TB free egress per month via CloudFront's free tier**

For subsequent requests for the same output (e.g. the same archive requested twice), the output already exists in S3 and the request becomes a standard redirect — no compute needed.

The full flow for a normal vs. compute request:

```
Normal request:
Client → OpenMediaCloud → 307 redirect → R2(free egress) / S3(1TB free egress assuming CloudFront is used)

Compute request (first time):
Client → OpenMediaCloud → EC2 pulls sources from R2 (free ingress)
                       → EC2 computes output
                       → EC2 writes output to S3 (free intra-region, see below)
                       → 307 redirect → CloudFront → S3 (1 TB free egress)

Compute request (cached):
Client → OpenMediaCloud → 307 redirect → CloudFront → S3 (1 TB free egress)
```

---

## Caveats

### 1. Free EC2 → S3 transfer is NOT automatic — it requires a VPC Gateway Endpoint

The claim that EC2-to-S3 transfer within the same region is free is true, but **only when the traffic stays within AWS's internal network**. By default, if your EC2 instance accesses S3 using its public IP (which is the default without additional configuration), the traffic goes over the public internet and AWS charges for it.

To keep the traffic internal and free, you must configure an **S3 VPC Gateway Endpoint** in your VPC:

- Navigate to **VPC → Endpoints → Create Endpoint** in the AWS Console
- Select service type **AWS Services**, find `com.amazonaws.<your-region>.s3`
- Associate it with the VPC and subnet your EC2 instance is in
- AWS automatically updates the route table to direct all S3-bound traffic through the endpoint

Once this is done, all EC2 → S3 traffic in the same region is routed over AWS's internal backbone network, never touches the public internet, and incurs no data transfer charge. The Gateway Endpoint itself is free — no hourly fee and no per-GB processing charge.

Without this setup, the "free intra-region transfer" assumption in the workaround above does not hold.

### 2. EC2 and S3 must be in the same AWS region

The VPC Gateway Endpoint only works for S3 buckets in the same region as your VPC. If your EC2 instance is in `us-east-1` but your S3 bucket is in `eu-west-1`, the free transfer does not apply regardless of the VPC endpoint configuration. Cross-region EC2-to-S3 transfers are billed at standard data transfer rates.

This means the workaround requires that you deliberately place your S3 bucket in the same AWS region as your EC2 instance. This is a setup decision you must make when provisioning your infrastructure — it cannot be fixed after the fact without migrating your bucket.

### 3. This workaround only applies to R2 + EC2 + AWS S3 + CloudFront

If your setup uses EC2 with **Cloudflare R2** instead of AWS S3 for caching and storing the computed output, the free intra-region transfer benefit does not apply. R2 is on Cloudflare's network, not AWS's. Writing a computed output from EC2 to R2 crosses network boundaries and will incur AWS outbound data transfer charges, the same as serving it directly to the client.

For the EC2 + R2 configuration, the compute workaround still functions technically, but the economics are different. You pay to write the output from EC2 to R2. However, you pay this cost once and then every subsequent request for the same output is served by R2 at zero egress cost. Whether this is cost-effective depends on how frequently the same output is requested. For one-time archive downloads it is likely not worth it; for a transcoded video that many users will watch, amortised over requests it can still come out ahead.

### 4. Output caching strategy

The compute workaround only makes economic sense if outputs are cached in S3 rather than regenerated on every request. The implementation will use a deterministic key derived from the request parameters (e.g. a hash of the asset IDs for an archive, or the asset ID plus transcode parameters for a video) to check whether the output already exists in S3 before doing any work. If it exists, the request is a standard redirect. If it does not, computation runs, the output is saved to S3, and only then is the redirect issued.

Outputs are not deleted automatically. This means storage costs grow over time as more unique outputs are generated and cached. You may want to configure an S3 lifecycle rule to expire outputs after a suitable period (e.g. 7 or 30 days) if storage cost is a concern.

### 5. This is not yet implemented

The workaround described here is a plan, not a shipping feature. It requires coordinating the compute pipeline, the S3 write, the cache check, and the redirect logic — and it depends on infrastructure that not all users will have configured (VPC Gateway Endpoint, same-region bucket, CloudFront distribution).

For affected requests such as archive downloads, the current behaviour is to pass them through to the upstream server and let the upstream handle them natively, at the cost of EC2 egress.

Jellyfin transcoding is handled differently. Rather than passing the request through, OpenMediaCloud manipulates the playback info response to steer the client into requesting the original source file directly instead of a transcoded variant. The tradeoff is that clients incapable of playing the source format will fail to play the content entirely. This primarily affects the web client, which has limited codec and container support. Mobile apps and dedicated desktop players (SwiftFin, Jellyfin Media Player, Infuse) are significantly more resilient in this regard, as they rely on native platform media frameworks that support a much wider range of formats.

---

## Summary

| Scenario | EC2 → S3 free? | Client delivery | Notes |
|---|---|---|---|
| EC2 + S3 (same region) + CloudFront | Yes, with VPC Gateway Endpoint | CloudFront (1 TB free) | Optimal setup |
| EC2 + S3 (same region), no VPC endpoint | No — traffic goes over public internet | CloudFront (1 TB free) | Easy to fix: add the Gateway Endpoint |
| EC2 + S3 (different region) | No | CloudFront (1 TB free) | Requires bucket migration to fix |
| EC2 + R2 | No — cross-network transfer | R2 (free egress) | Pay once to write, then free delivery |
