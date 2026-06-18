# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.4] - 2026-06-18

### Added

- `configure opencode` now fetches `GET /v1/models/metadata` from the gateway
  and writes enriched model entries to `opencode.json`: context/input/output
  limits, input/output modalities, reasoning flag, and reasoning-effort
  variants (e.g. `low`, `medium`, `high`).

### Changed

- Falls back to plain `GET /v1/models` when the metadata endpoint is
  unavailable, preserving backward compatibility with older gateways.

## [0.0.3] - 2026-06-07

### Fixed

- `configure opencode` now reuses the API key stored by a previous run instead
  of always prompting for it, matching the `models` and `status` commands.

## [0.0.2] - 2026-06-07

### Changed

- `voban status` now reads `GET /v1/spend` with the `sk-sov-` API key instead
  of `/api/me` and `/api/me/spend`, which require a Zitadel JWT. The command no
  longer prints the email or customer fields; it reports the user, spend,
  budget, and blocked state.

### Removed

- The unused `Me()` method from the internal HTTP client.

## [0.0.1] - 2026-06-07

### Added

- Initial `voban` CLI with `configure opencode`, `models`, and `status`
  commands targeting the Voban gateway.

[0.0.4]: https://github.com/vobanai/voban-cli/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/vobanai/voban-cli/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/vobanai/voban-cli/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/vobanai/voban-cli/releases/tag/v0.0.1
