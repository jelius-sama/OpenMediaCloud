# Setup Guide

This guide walks you through setting up OpenMediaCloud alongside Jellyfin on a Linux server. It assumes you are comfortable with the command line and have basic Linux administration knowledge.

---

## Prerequisites

Make sure the following are installed and configured before proceeding:

- **rclone** — configured with your storage backend (R2, S3, or any S3-compatible provider). Refer to the [rclone documentation](https://rclone.org/docs/) for remote setup. The remote used in the service files is named `s3` — adjust to match your configured remote name.
- **Native Jellyfin OR Docker and Docker Compose** — for running Jellyfin Server. 
- **Caddy** (or any reverse proxy) — for TLS termination. Install via your package manager (`apt`, `dnf`, etc.) or from [caddyserver.com](https://caddyserver.com).
- **fuse3** — required for rclone FUSE mounts (`sudo apt install fuse3` or equivalent).

---

## Step 1 — Download the binary

Download the latest OpenMediaCloud binary for your OS and architecture from the [releases page](https://github.com/jelius-sama/OpenMediaCloud/releases).

Binaries are named in the format `OpenMediaCloud-{os}-{arch}`, for example:
- `OpenMediaCloud-linux-amd64`
- `OpenMediaCloud-linux-arm64`

Move it to `/usr/local/bin` and make it executable:

```sh
sudo mv OpenMediaCloud-linux-amd64 /usr/local/bin/OpenMediaCloud
sudo chmod +x /usr/local/bin/OpenMediaCloud
```

---

## Step 2 — Create the config directory and environment file

OpenMediaCloud reads its configuration from `~/.config/OpenMediaCloud/.env` OR `/etc/OpenMediaCloud/.env`.

```sh
mkdir -p ~/.config/OpenMediaCloud
vim ~/.config/OpenMediaCloud/.env
```

Paste and fill in the following:

```sh
JELLYFIN_HOST="http://localhost:8096"
JELLYFIN_API_KEY="your-jellyfin-api-key"
JELLYFIN_USER_ID="your-jellyfin-user-id"

ACCESS_KEY_ID="your-access-key-id"
SECRET_ACCESS_KEY="your-secret-access-key"
BUCKET_NAME="your-bucket-name"

# When using Cloudflare R2, set this to your R2 endpoint.
# Leave empty when using AWS S3 directly.
BASE_URL="https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com"

# Set to "auto" for R2, or your AWS region code (e.g. "us-east-1") for S3.
AWS_REGION="auto"

# CloudFront (optional, AWS S3 users only)
# Set your CloudFront distribution domain to enable CloudFront mode.
CLOUDFRONT_ENDPOINT=""

# Only required if using signed URLs with CloudFront.
CLOUDFRONT_KEY_PAIR_ID=""
# Must be an absolute path. The file itself can be anywhere on the filesystem.
CLOUDFRONT_PRIVATE_KEY_PATH="/home/your-user/.config/OpenMediaCloud/private_key.pem"

# For multi-service configuration you can configure the router using the config below.
UPSTREAM_JELLYFIN_HOST=tv.example.com
UPSTREAM_IMMICH_HOST=photos.example.com
UPSTREAM_KOMGA_HOST=manga.example.com
```

> **Where to find these values:**
> - `JELLYFIN_API_KEY` — Jellyfin dashboard → Admin → API Keys → create a new key
> - `JELLYFIN_USER_ID` — visible in the URL when editing a user in the Jellyfin dashboard, or from the Jellyfin API
> - `ACCESS_KEY_ID` / `SECRET_ACCESS_KEY` — from your R2 or S3 bucket's API credentials page
> - `CLOUDFRONT_KEY_PAIR_ID` — CloudFront console → Key management → Public keys

If using CloudFront signed URLs, generate your RSA key pair and place the files wherever you prefer on the filesystem — the location does not matter as long as `CLOUDFRONT_PRIVATE_KEY_PATH` points to it with an absolute path:
```sh
openssl genrsa -out private_key.pem 2048
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

> **Important:** AWS CloudFront only accepts 2048-bit RSA keys. Do not use 4096-bit — it will be rejected when you upload the public key to the CloudFront console.

Upload `public_key.pem` to the CloudFront console under Key management → Public keys. Keep `private_key.pem` on your server and set `CLOUDFRONT_PRIVATE_KEY_PATH` to its absolute path.

Be mindful of file permissions on the private key — if you restrict read access with `chmod 600`, make sure the user that OpenMediaCloud runs as (set in `OpenMediaCloud.service`) is the owner of that file, otherwise the service will fail to read it.

---

## Step 3 — Install the systemd service files

Write the provided service files to `/etc/systemd/system/`:

```sh
sudo vim /etc/systemd/system/rclone-media.service
sudo vim /etc/systemd/system/OpenMediaCloud.service
```

**[`rclone-media.service`](./rclone-media.service)** — mounts your storage bucket to `/mnt/media` using rclone FUSE. Edit the `ExecStart` line to match your rclone remote name and bucket:

```ini
ExecStart=/usr/bin/rclone mount s3:your-media-bucket /mnt/media \
    --allow-other \
    --uid=1000 \
    --gid=1000 \
    --vfs-cache-mode writes \
    --cache-dir /mnt/ebs/rclone-cache \
    --vfs-cache-max-size 4G \
    --vfs-cache-max-age 30m
```

Replace `s3:your-media-bucket` with your configured rclone remote and bucket name. Adjust `--cache-dir` to a suitable path on your server.

**[`OpenMediaCloud.service`](./OpenMediaCloud.service)** — runs the OpenMediaCloud proxy. Edit the `User` and `Group` fields to match your server user:

```ini
User=your-username
Group=your-username
```

---

## Step 4 — Configure Docker or Native Jellyfin to start after the rclone mount

Docker or Native Jellyfin must start after the rclone mount is ready, otherwise Jellyfin will start with an empty media directory.

Create the override directory and copy the provided config (setup for native jellyfin should also be similar):

```sh
sudo mkdir -p /etc/systemd/system/docker.service.d
sudo cp override.conf /etc/systemd/system/docker.service.d/
```

**[`docker.service.d/override.conf`](./docker.service.d/override.conf)** adds `rclone-media.service` as a dependency of Docker so systemd starts them in the correct order.

---

## Step 5 — Set up Jellyfin with Docker Compose

Create a directory for your Jellyfin setup and copy the provided compose file:

```sh
mkdir -p ~/jellyfin
cp docker-compose.yml ~/jellyfin/
```

**[`docker-compose.yml`](./docker-compose.yml)** — edit the volume mounts to match your media directory structure. Each bind mount maps a directory from your rclone-mounted `/mnt/media` into the Jellyfin container. The `target` path inside the container must match your S3 bucket folder names exactly — see the [Storage Path Requirements](../README.md#storage-path-requirements) section in the main README.

Example:
```yaml
volumes:
  - type: bind
    source: /mnt/media/AMVs   # path on your server (inside rclone mount)
    target: /AMVs              # path Jellyfin sees — must match bucket folder name
    read_only: true
```

---

## Step 6 — Enable and start services

Reload systemd to pick up the new service files, then enable and start everything:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now rclone-media.service
sudo systemctl enable --now OpenMediaCloud.service
```

Verify the rclone mount is working before starting Jellyfin:

```sh
ls /mnt/media
```

Then start Jellyfin:

```sh
cd ~/jellyfin && docker compose up -d
```

---

## Step 7 — Configure your reverse proxy

Point your reverse proxy at OpenMediaCloud (default port `8000`), not directly at Jellyfin.

Example Caddy configuration:

```
tv.yourdomain.com {
    reverse_proxy 127.0.0.1:8000
}
```

OpenMediaCloud will forward all non-media requests to Jellyfin internally.

---

## Verifying the setup

Check that all services are running:

```sh
sudo systemctl status rclone-media.service
sudo systemctl status OpenMediaCloud.service
sudo systemctl status docker
```

Check OpenMediaCloud logs to confirm it is intercepting media requests:

```sh
sudo journalctl -u OpenMediaCloud.service -f
```

When you play a video you should see log lines like:

```
[OK] Caught video request: /Videos/{id}/stream.mp4
[OK] Redirected client to S3 for object: AMVs/your-video.mp4
```

If you see these, media is being served directly from your storage bucket and your server is no longer in the data path.

If anything looks off despite following the setup guide, clone the repository and build with the debug flag — debug builds emit detailed logs including extracted item IDs, resolved file paths, generated presigned URLs, and auth validation results, which makes it much easier to pinpoint where things are going wrong:

```sh
git clone https://github.com/jelius-sama/OpenMediaCloud.git
cd OpenMediaCloud
make run
```

> If you are on your local machine and would like to copy the debug binary to your server (using scp), edit the Makefile to make sure that the `build` recipe is prefixed with the right OS and architecture of your server, if the server OS and architecture is same as your local machine then this step can be skipped. Then run `make build` which will produce a debug binary at `./bin/OpenMediaCloud` just copy it to your server.
