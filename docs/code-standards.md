# Code Standards

Guidelines for contributors. Short, opinionated, Go-idiomatic.

## Layout

- `cmd/` — Cobra subcommands. One subcommand per file or per small group.
- `internal/` — Application infrastructure that must not be imported by
  external consumers.
- `pkg/tools` — The single public extension point. New tools live here.
- `legacy/` — Reserved for the pre-Go TypeScript source during the port
  only. Do not add new files here; remove the directory once the port
  is considered complete.

## Naming

- Binary, module, config dir, env prefix all share the same brand:
  `haravan-cli`, `github.com/pluginmd/haravan-cli`, `~/.haravan-cli/`,
  `HARAVAN_*`.
- Tool `Name` is the MCP identifier verbatim (`haravan_orders_list`,
  `hrv_orders_summary`). The cobra subcommand slug is derived
  automatically by stripping the `haravan_`/`hrv_` prefix.

## Error handling

- Return errors, don't log-and-swallow.
- Classify HTTP errors at the client edge; call sites discriminate with
  `errors.As` against the typed wrappers in `internal/client/errors.go`.
- `client.FriendlyMessage(err)` produces the user-facing text used by
  both CLI output and MCP error results.

## JSON I/O

- Tool handlers should return `tools.Result{Data: json.RawMessage}`
  whenever the response is structured; the framework pretty-prints
  it for the CLI and emits it verbatim for MCP.
- Prefer `json.RawMessage` over intermediate typed structs when the
  tool's job is to pass through the Haravan envelope unchanged.
- Use `tools.EnvelopeWrap(body, "order")` for create/update bodies
  — it accepts either the bare resource object or the already-wrapped
  envelope so CLI users don't have to remember.

## Dependencies

Keep the dependency graph flat:

```
cmd/            → internal/* + pkg/tools
internal/*      → other internal/* (no cmd, no pkg)
pkg/tools       → internal/client, internal/logger, internal/iostreams
pkg/tools/<dom> → pkg/tools only
```

Any new tool should only import `pkg/tools` and std lib. If you feel
the urge to reach into `internal/client` from a tool handler, promote
whatever helper you need to `pkg/tools` instead.

## Tests

- Unit tests live next to the code under test.
- Use `httptest.NewServer` for HTTP client tests rather than mocking.
- Use `t.TempDir()` + `t.Setenv("HARAVAN_CLI_HOME", ...)` to isolate
  config / token-store tests.
- Enforce registry invariants with a test (see
  `pkg/tools/registry_test.go`) — duplicate names or nil handlers
  are bugs, not features.

## Commits

Conventional commits (`feat:`, `fix:`, `test:`, `chore:`, `docs:`).
Scope the change (`feat(phase-5b): port orders domain`). Co-author
trailers are welcome.

## Formatting

- `gofmt` + `goimports` (enforced by golangci-lint).
- 120-column soft limit. Don't reformat unrelated lines.
- Comments follow godoc conventions: first sentence is a full sentence
  starting with the identifier name.
