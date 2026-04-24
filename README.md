# haravan-cli

Unified Go CLI and Model Context Protocol (MCP) server for the Haravan
e-commerce platform. One binary, two surfaces:

- **CLI** — 70 subcommands across 8 domains for scripting and debugging.
- **MCP server** — every command is also exposed as an MCP tool over
  stdio or HTTP/SSE, ready for Claude.ai, Claude Code, Cursor, and any
  other MCP-capable client.

No Node, no npm, no runtime dependencies — a single static binary (~15 MB).

---

## Install

### `go install`

```bash
go install github.com/pluginmd/haravan-cli@latest
```

### From source

```bash
git clone https://github.com/pluginmd/haravan-cli
cd haravan-cli
make install        # installs to /usr/local/bin/haravan-cli
```

### Docker

```bash
cp docker/.env.example docker/.env
# Edit docker/.env and set HARAVAN_ACCESS_TOKEN
docker compose -f docker/docker-compose.yml up -d
```

The container exposes MCP over HTTP/SSE on `:4567`.

---

## Configure

Two credential flows are supported.

### Private app token (simplest)

```bash
export HARAVAN_ACCESS_TOKEN=...
haravan-cli shop get
```

### OAuth 2.0

```bash
haravan-cli config set --app-id=YOUR_APP_ID --app-secret=YOUR_APP_SECRET
haravan-cli auth login                  # opens browser, stores token
haravan-cli auth status                 # inspect stored tokens
```

Config lives at `~/.haravan-cli/` (override with `HARAVAN_CLI_HOME`):

```
~/.haravan-cli/
├── config.json     # api_base, app_id, app_secret
└── tokens.json     # keyed by app_id, 0600 perms, atomic writes
```

### Resolution order

For every API call the token is resolved in this order:

1. `--token` flag
2. `HARAVAN_ACCESS_TOKEN` env
3. Stored OAuth token for the effective `app_id`
   (auto-refresh when expired + refresh token present)

---

## CLI usage

Top-level commands:

```text
haravan-cli
├── auth              login, logout, status
├── config            show, set, path
├── mcp               serve (stdio | HTTP/SSE), tools
├── orders            13 tools: list, get, create, confirm, close, cancel, …
├── products          11 tools: list, get, create, variants, …
├── customers         14 tools: list, search, groups, addresses, …
├── inventory         5  tools: adjustments, adjust_or_set, locations
├── shop              6  tools: shop info, locations, users, shipping_rates
├── content           11 tools: pages, blogs, articles, script_tags
├── webhooks          3  tools: list, subscribe, unsubscribe
└── smart             7  tools: orders_summary, top_products, rfm, …
```

Every tool is both a shell subcommand and an MCP tool. Handler code is
written once and dispatched from whichever surface called it.

### Examples

```bash
# Shop info
haravan-cli shop get

# List today's orders (with auto-pagination)
haravan-cli orders list --status=any --created_at_min=2026-04-24T00:00:00Z --fetch_all

# Create an order from a file
haravan-cli orders create --body=@order.json

# Smart: 30-day summary with prior-period comparison
haravan-cli smart orders_summary

# RFM customer segmentation
haravan-cli smart customer_segments --min_orders=1

# List everything the MCP server will advertise
haravan-cli mcp tools
```

JSON bodies for create/update commands accept either inline JSON or
`@path/to/file.json`, and the outer envelope (`{"order": …}`,
`{"product": …}`, etc.) is added automatically if you pass the bare
object.

---

## MCP server

Start a stdio server (the default an MCP client expects):

```bash
haravan-cli mcp serve
```

Start an HTTP/SSE server for network-accessible integrations:

```bash
haravan-cli mcp serve --http --addr=:4567
```

### Claude Code / Desktop configuration

```json
{
  "mcpServers": {
    "haravan": {
      "command": "haravan-cli",
      "args": ["mcp", "serve"],
      "env": {
        "HARAVAN_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### Cursor (`~/.cursor/mcp.json`)

Same shape as above.

### Claude Skill

The `claudeskill/haravan-mcp/` directory contains a reasoning layer
designed for Claude.ai. It documents every tool, the decision tree
for which tool to use when, and Vietnam-specific e-commerce
benchmarks. Upload the folder to Claude.ai as a Skill or copy it to
`~/.claude/skills/haravan-mcp/` for Claude Code.

---

## Environment variables

| Variable | Purpose |
|----------|---------|
| `HARAVAN_ACCESS_TOKEN` | Private app token |
| `HARAVAN_APP_ID` | OAuth client_id |
| `HARAVAN_APP_SECRET` | OAuth client_secret |
| `HARAVAN_API_BASE` | Override main API base URL |
| `HARAVAN_WEBHOOK_BASE` | Override webhook API base URL |
| `HARAVAN_CLI_HOME` | Override config/token directory |
| `HARAVAN_LOG_LEVEL` | `debug` / `info` / `warn` / `error` / `off` |

---

## Development

```bash
make build               # go build with version injection
make test                # go test -race ./...
make vet                 # go vet ./...
make lint                # requires golangci-lint
make release-snapshot    # local goreleaser snapshot
```

Architecture, design decisions, and contribution guidance live in
[docs/](docs/).

---

## Licence

MIT — see [LICENSE](LICENSE).
