# AGENTS.md

`voban` is a small Go CLI (Go 1.26.1, **zero third-party dependencies** — keep it stdlib-only). It configures AI coding tools (currently only opencode) to use the Voban gateway.

## Commands

Run in this order; CI (`.github/workflows/ci.yml`) runs vet + race tests on Linux/macOS/Windows.

```bash
go vet ./...
go test -race ./...                                  # CI uses -race; keep it
go build -o voban ./cmd/voban
go test -race ./internal/opencode -run TestName      # single test
```

`./voban` and `*.test` are gitignored. Release binaries are built by tag push (`v*`) in `release.yml`.

## Architecture

- `cmd/voban/main.go` — entrypoint. Hand-rolled arg dispatch (`configure opencode` / `models` / `status`); **no cobra/flag library**. Key resolution: `VOBAN_API_KEY` -> stored opencode key -> interactive prompt.
- `internal/config` — `BaseURL()` (env `VOBAN_BASE_URL`, default `https://api.voban.ai`) and `ValidateAPIKey` (must start with `sk-sov-`).
- `internal/client` — minimal HTTP client for `GET /v1/spend`, `/v1/models`. Bearer auth.
- `internal/opencode` — writes opencode's `opencode.json` (provider) and `auth.json` (key).

## Gotchas (do not break)

- `paths.go` deliberately replicates opencode's `xdg-basedir`: `~/.config` and `~/.local/share` on **every** platform incl. macOS/Windows, honoring `XDG_CONFIG_HOME`/`XDG_DATA_HOME`. **Do NOT switch to `os.UserConfigDir`** — that would diverge from opencode and write to the wrong place.
- Config/auth writes are **merge-only**: only voban-owned entries are touched (`provider.voban` in config, `voban` in auth). All other user config must be preserved.
- `auth.json` is written `0600` (holds the secret key); `opencode.json` is `0644`.
- `readJSONObject` treats a missing or corrupt file as an empty object by design; permission/I/O errors must fail loud.

## Conventions

- Tests use `net/http/httptest` and `t.TempDir()`; one scenario per test (parameterized where it fits).
- Wrap errors with `%w` and context; never swallow them.
