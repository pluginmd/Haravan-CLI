# System Architecture

`haravan-cli` is a single Go binary that serves two surfaces — a CLI and
an MCP server — from one registry of tool definitions.

```
                      ┌─────────────────────────────┐
                      │        haravan-cli          │
                      │   (single Go binary)        │
                      └──────────────┬──────────────┘
                                     │
                ┌────────────────────┴────────────────────┐
                │                                         │
        ┌───────▼────────┐                       ┌────────▼────────┐
        │   Cobra CLI    │                       │   MCP server    │
        │  (subcommands) │                       │ (stdio + SSE)   │
        └───────┬────────┘                       └────────┬────────┘
                │                                         │
                └────────────┬────────────────────────────┘
                             │
                   ┌─────────▼──────────┐
                   │  tools.Registry    │      70 Tool definitions —
                   │  (process-wide)    │      one Handler each.
                   └─────────┬──────────┘
                             │
                   ┌─────────▼──────────┐
                   │  internal/client    │      HTTP, auth, rate-limit,
                   │   (HaravanClient)   │      typed errors, pagination.
                   └─────────┬──────────┘
                             │
                        apis.haravan.com
                        webhook.haravan.com
```

---

## Package map

| Package | Role |
|---------|------|
| `main`, `cmd/` | Cobra command tree, DI factory wiring |
| `internal/config` | `~/.haravan-cli/config.json` persistence, env var map |
| `internal/auth` | Token store, OAuth 2.0 authorization-code flow, token resolver |
| `internal/client` | `Client` with middleware-free design: Bearer auth, JSON enc/dec, typed errors, leaky-bucket rate limit, `PaginateAll` helper |
| `internal/cmdutil` | Factory (`IOStreams`, `Config`), persistent auth flags, `BuildClient` / `BuildWebhookClient` |
| `internal/mcpserver` | Wraps every registered Tool as an `mcp.Tool`; stdio + HTTP/SSE transports |
| `internal/iostreams` | stdin/stdout/stderr DI |
| `internal/logger` | Leveled logger writing to stderr |
| `internal/build` | `ldflags`-injected version/commit/date |
| `pkg/tools` | `Tool`, `Flag`, `Input`, `Result`, `Registry` — the core abstraction |
| `pkg/tools/<domain>` | One file per domain, registers tools via `init()` |

---

## Tool lifecycle

Every Haravan endpoint is modeled as **one** `tools.Tool`:

```go
type Tool struct {
    Name     string      // e.g. "haravan_orders_list"
    Short    string
    Long     string
    Category Category    // orders | products | customers | …
    Scopes   []string    // advisory OAuth scopes
    Flags    []Flag      // drive both cobra and JSON Schema
    Handler  func(ctx, *Deps, Input) (*Result, error)
}
```

Registration happens at `init()` time in each `pkg/tools/<domain>/*.go`
file. The root command imports every domain package with a blank
identifier so `init()` runs, then `tools.MountAll` mounts every
registered tool as a cobra subcommand under its category.

The MCP server walks the same registry, produces a JSON Schema from
each tool's `Flags` via `BuildJSONSchema`, and registers a handler
adapter that:

1. Decodes the MCP call arguments into `tools.Input`
2. Invokes `Tool.Handler` with shared `*Deps`
3. Emits the `Result` as either a JSON text content block or a plain
   text content block (or an error result)

Both surfaces share the same `Deps` — auth resolution happens once at
command/server startup, not per call.

---

## HTTP client design

`internal/client/Client` intentionally stops at raw HTTP + response
envelope + typed errors. The TS legacy used a middleware chain
(validation → rate-limit → pagination → error-handler); the Go port
inlines those concerns into explicit, testable primitives:

- **Auth** — `Authorization: Bearer <token>` added to every request.
- **Rate limit** — `X-Haravan-Api-Call-Limit` parsed into a
  per-client leaky bucket. Upcoming requests block (with
  `ctx.Done()` honoring) when the estimated fill passes a safety
  threshold.
- **Errors** — non-2xx responses produce an `*APIError` wrapped in a
  status-specific typed error (`*UnauthorizedError`,
  `*ForbiddenError`, `*NotFoundError`, `*ValidationError`,
  `*RateLimitError`, `*ServerError`). Callers use `errors.As`.
- **Pagination** — `PaginateAll` is a top-level helper, not a
  middleware, so tools can opt in per call via `--fetch_all`.

Validation in the TS port was Zod + schema middleware; in Go we lean
on:

- Cobra for CLI flag parsing
- JSON Schema (advertised to MCP clients) for declarative contracts
- Explicit checks in each handler where business rules apply

---

## Auth resolution

`internal/auth/Resolver` is the single source of truth:

1. Explicit `--token` flag or `Resolver.Token`
2. `HARAVAN_ACCESS_TOKEN` env
3. Stored OAuth token for the effective `app_id`, with automatic
   refresh when expired and a refresh token + app secret are
   available

The token store (`~/.haravan-cli/tokens.json`) writes atomically with
0600 perms, keyed by `app_id` so multiple apps can coexist.

---

## MCP dispatch

`internal/mcpserver/server.go`:

- Registers each Tool with `mcp.WithRawInputSchema(...)` so the
  server advertises the same JSON Schema that `BuildJSONSchema`
  produces for CLI help — a single source of truth for contracts.
- The adapter catches Handler errors and routes them through
  `client.FriendlyMessage` so MCP clients see actionable text
  ("Authentication failed — run `haravan-cli auth login`") rather
  than raw HTTP status strings.
- Transports: `ServeStdio` (default, what Claude Desktop/Cursor
  expect) and `NewSSEServer(addr)` (for network-accessible
  deployments; the Docker image runs this).

---

## Webhook split

Webhook subscription endpoints live on a different host
(`webhook.haravan.com`) from the rest of the API. `Deps.WebhookClient`
is built eagerly at startup with the same token but a different base
URL; only the three webhook tools reach for it.

---

## Extending

Adding a new tool is mechanical:

1. Drop a `.go` file under `pkg/tools/<domain>/`
2. Call `tools.Register(&tools.Tool{...})` in `init()`
3. Done — it appears as both a cobra subcommand and an MCP tool
   automatically

No handler code needs to know anything about cobra or MCP. Tests at
`pkg/tools/registry_test.go` enforce that every tool has a non-nil
handler, and `internal/mcpserver/server_test.go` asserts the total
count matches expectations.
