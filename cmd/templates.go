package main

import (
    "fmt"
    "io"
)

const ENV = `JELLYFIN_HOST="http://localhost:8096"
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
`

const SERVICE = `[Unit]
Description=Specialized Proxy for Jellyfin
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/OpenMediaCloud
Restart=always
User=ec2-user
Group=ec2-user

[Install]
WantedBy=multi-user.target
`

func GlobalHelp(w io.Writer) {
    const (
        reset  = "\033[0m"
        bold   = "\033[1m"
        dim    = "\033[2m"
        cyan   = "\033[36m"
        green  = "\033[32m"
        yellow = "\033[33m"
        white  = "\033[97m"
    )

    fmt.Fprint(w, cyan+bold+`
  ██████╗ ██████╗ ███████╗███╗   ██╗    ███╗   ███╗███████╗██████╗ ██╗ █████╗      
 ██╔═══██╗██╔══██╗██╔════╝████╗  ██║    ████╗ ████║██╔════╝██╔══██╗██║██╔══██╗     
 ██║   ██║██████╔╝█████╗  ██╔██╗ ██║    ██╔████╔██║█████╗  ██║  ██║██║███████║     
 ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║    ██║╚██╔╝██║██╔══╝  ██║  ██║██║██╔══██║     
 ╚██████╔╝██║     ███████╗██║ ╚████║    ██║ ╚═╝ ██║███████╗██████╔╝██║██║  ██║     
  ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝    ╚═╝     ╚═╝╚══════╝╚═════╝ ╚═╝╚═╝  ╚═╝     
                                                                                     
  ██████╗██╗      ██████╗ ██╗   ██╗██████╗                                          
 ██╔════╝██║     ██╔═══██╗██║   ██║██╔══██╗                                         
 ██║     ██║     ██║   ██║██║   ██║██║  ██║                                         
 ██║     ██║     ██║   ██║██║   ██║██║  ██║                                         
 ╚██████╗███████╗╚██████╔╝╚██████╔╝██████╔╝                                         
  ╚═════╝╚══════╝ ╚═════╝  ╚═════╝ ╚═════╝                                          
`+reset+`
`+dim+`  A lightweight proxy to reduce cloud hosting costs.`+reset+`
`+dim+`  Redirects media requests directly to R2/S3, bypassing your server entirely.`+reset+`

`+bold+white+`USAGE`+reset+`
  `+cyan+`OpenMediaCloud`+reset+`                        Start the proxy server
  `+cyan+`OpenMediaCloud`+reset+` [options]              Start with options
  `+cyan+`OpenMediaCloud`+reset+` gen <subcommand>       Generate a template file

`+bold+white+`COMMANDS`+reset+`
  `+green+bold+`gen env`+reset+`                              Print a .env template to stdout
  `+green+bold+`gen env -o <path>`+reset+`                   Write .env template to a file
  `+green+bold+`gen service`+reset+`                         Print a systemd service template to stdout
  `+green+bold+`gen service -o <path>`+reset+`               Write systemd service template to a file

`+bold+white+`OPTIONS`+reset+`
  `+yellow+`-env`+reset+`    <path>                       Load environment variables from a custom path
  `+yellow+`-version`+reset+`                             Show version and build info

`)
}

