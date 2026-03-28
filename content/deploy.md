---
section: guide
---
# Deploy

## Single Binary

Compile and copy anywhere:

```bash
kilnx build app.kilnx -o myapp
scp myapp server:~/
ssh server './myapp --port 3000 --db /data/app.db'
```

The binary contains everything: HTTP server, htmx.js, SQLite, bcrypt, your app code. No Docker, no Node, no Python, no runtime.

## Railway

One-click deploy:

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/kilnx)

Or manually:

1. Fork [kilnx-org/railway-template](https://github.com/kilnx-org/railway-template)
2. Connect to Railway
3. Edit `app.kilnx`, push, Railway rebuilds automatically

Attach a [Railway Volume](https://docs.railway.com/guides/volumes) at `/data` for persistent SQLite.

## Docker

```dockerfile
FROM ghcr.io/kilnx-org/kilnx:0.1.0 AS builder
COPY app.kilnx /kilnx/app.kilnx
RUN kilnx build app.kilnx -o /app/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/server /usr/local/bin/server
CMD ["server"]
```

```bash
docker build -t myapp .
docker run -p 8080:8080 -v ./data:/data myapp
```

## Fly.io

```bash
flyctl launch
flyctl deploy
```

## Bare Metal (systemd)

```ini
[Unit]
Description=My Kilnx App
After=network.target

[Service]
ExecStart=/usr/local/bin/myapp
WorkingDirectory=/data
Restart=always
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

```bash
sudo cp myapp.service /etc/systemd/system/
sudo systemctl enable --now myapp
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `sqlite://app.db` | SQLite database path |
| `SECRET_KEY` | (none) | Session signing secret (required for auth) |
