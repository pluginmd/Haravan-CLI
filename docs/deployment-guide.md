# Deployment Guide

Three supported ways to run `haravan-cli` in production.

## 1. Local binary (development, one-shot scripts)

```bash
go install github.com/pluginmd/haravan-cli@latest
haravan-cli mcp serve
```

The stdio MCP server runs in-process. Claude Desktop / Cursor /
Claude Code spawn it automatically — they don't need a separate
daemon.

## 2. Long-running HTTP/SSE server (shared/multi-user)

Build the binary and run it as a systemd unit (or equivalent):

```ini
[Unit]
Description=Haravan MCP Server
After=network.target

[Service]
User=haravan
Environment=HARAVAN_ACCESS_TOKEN=...
ExecStart=/usr/local/bin/haravan-cli mcp serve --http --addr=:4567
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Point any MCP client at `http://<host>:4567/sse`.

## 3. Docker

```bash
cp docker/.env.example docker/.env
# Fill in HARAVAN_ACCESS_TOKEN
docker compose -f docker/docker-compose.yml up -d
```

The image is multi-stage Go (build from source) on Alpine:

- Base: `alpine:3.20`
- Size: ~15 MB
- Runs as unprivileged user `app`
- `tini` as PID 1 for clean signal handling
- Default command: `mcp serve --http --addr=:4567`

### Behind a reverse proxy

nginx:

```nginx
location /haravan-mcp/ {
    proxy_pass http://127.0.0.1:4567/;
    proxy_http_version 1.1;
    proxy_set_header Connection "";      # long-lived SSE
    proxy_buffering off;
    proxy_read_timeout 3600s;
}
```

Pass `--base-url=https://public-host/haravan-mcp` to the server so
the SSE endpoint URLs it advertises to clients are correct.

### Cloudflare Tunnel

```bash
cloudflared tunnel --url http://localhost:4567
```

## Health

- No dedicated `/health` endpoint yet — SSE `GET /sse` returns 200
  when the server is up; `docker-compose.yml` uses it as the
  healthcheck.
- Add `HARAVAN_LOG_LEVEL=debug` to see per-request logging on
  stderr. Ship to your log aggregator of choice.

## Secrets

- Never bake `HARAVAN_ACCESS_TOKEN` into an image; pass it via
  env at runtime (Docker `--env-file`, Kubernetes `Secret`, systemd
  `EnvironmentFile`).
- Config persisted under `~/.haravan-cli/tokens.json` is 0600 and
  atomic-write. Mount this directory as a tmpfs or a dedicated
  volume when running in a container so tokens survive restarts.

## Upgrade

```bash
# Binary install
go install github.com/pluginmd/haravan-cli@latest

# Docker
docker compose pull && docker compose up -d

# Source
git pull && make install
```

Config format is forward-compatible within a minor version.
Breaking config changes are noted in the CHANGELOG.
