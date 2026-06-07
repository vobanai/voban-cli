# voban CLI

`voban` configures AI coding tools to use the [Voban](https://voban.ai) gateway,
an OpenAI-compatible API with budget enforcement and the Voban Guard.

Today it configures [opencode](https://opencode.ai). More tools will follow.

## Install

Download a binary from the [releases page](https://github.com/vobanai/voban-cli/releases),
or build from source:

```bash
go install github.com/vobanai/voban-cli/cmd/voban@latest
```

## Usage

You need a Voban API key (it starts with `sk-sov-`). Create one from the Voban
web UI, then:

```bash
voban configure opencode
```

This:

1. Validates your key against the gateway and lists the models you can use.
2. Adds a `voban` provider to opencode's config (`opencode.json`).
3. Stores your key in opencode's `auth.json` (with `0600` permissions).

It never overwrites unrelated opencode settings, and it does not pin a default
model. Start opencode and run `/models` to pick a Voban model.

### Providing the key

The key is resolved in this order:

1. `VOBAN_API_KEY` environment variable
2. The key from a previous `voban configure opencode` run (for `models` and `status`)
3. An interactive prompt

```bash
VOBAN_API_KEY=sk-sov-... voban configure opencode
```

### Other commands

```bash
voban models   # list the models available to your key
voban status   # show your budget and spend
```

## Where opencode files are written

`voban` writes to opencode's global config and data directories. These follow
opencode's own resolution (the same `~/.config` and `~/.local/share` paths on
Linux, macOS, and Windows), honoring `XDG_CONFIG_HOME` and `XDG_DATA_HOME`:

| File | Default location |
|------|------------------|
| `opencode.json` | `$XDG_CONFIG_HOME/opencode/` or `~/.config/opencode/` |
| `auth.json` | `$XDG_DATA_HOME/opencode/` or `~/.local/share/opencode/` |

## Self-hosted gateways

Set `VOBAN_BASE_URL` to point at a self-hosted Voban gateway:

```bash
VOBAN_BASE_URL=https://gateway.internal.example voban configure opencode
```

The default is `https://api.voban.ai`.

## Development

```bash
go vet ./...
go test -race ./...
go build -o voban ./cmd/voban
```

## License

[MIT](LICENSE)
