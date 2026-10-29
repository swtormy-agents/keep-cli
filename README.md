# keep-cli

Agent-friendly CLI for [Keep](https://github.com/keephq/keep) (AIOps). One static binary, JSON on stdout, no MCP server required.

## Install

### Download a binary from GitHub Releases

1. Open [GitHub Releases](https://github.com/swtormy-agents/keep-cli/releases) and download the archive for your OS/arch (`keep_<version>_linux_amd64.tar.gz`, `darwin_arm64`, `windows_amd64.zip`, etc.).
2. Verify the SHA256 checksum against `checksums.txt` on the same release:

```bash
sha256sum -c checksums.txt --ignore-missing   # Linux
# or
shasum -a 256 -c checksums.txt                # macOS
```

3. Extract the archive. It contains `keep` (or `keep.exe`), `LICENSE`, `NOTICE`, and `THIRD_PARTY_NOTICES`.

Releases are published as **drafts** first; pick the latest draft or published release from the maintainer.

### Go install

```bash
go install github.com/swtormy-agents/keep-cli/cmd/keep@latest
```

### Build from source

```bash
git clone https://github.com/swtormy-agents/keep-cli.git
cd keep-cli
go build -o keep ./cmd/keep
```

## Authentication

Keep uses the `X-API-KEY` header (see [OpenAPI](docs/openapi.json)).

1. Create an API key in your Keep deployment (Settings → API keys).
2. Export it — **never pass the key on the command line** (it appears in shell history and `ps`):

```bash
export KEEP_API_KEY=your_key_here
export KEEP_API_URL=https://api.keephq.dev   # optional; this is the default
```

Alternatively, store credentials in a config file (mode `600`, directory mode `700`):

```json
~/.config/keep/config.json
{
  "api_url": "https://api.keephq.dev",
  "api_key": "your_key_here"
}
```

Optional `"access": "write"` in the same file enables writes without a TTY (lower precedence than `KEEP_ACCESS` and the `~/.config/keep/access` file). Prefer `keep access write` for interactive use.

Override the config file path with `KEEP_CONFIG` (single file). Set `KEEP_CONFIG_DIR` to relocate the whole config directory: `$KEEP_CONFIG_DIR/config.json` (API key and defaults) and `$KEEP_CONFIG_DIR/access` (write mode from `keep access write`). Default directory is `~/.config/keep/`. A project-local `.keep/config.json` is also read. On Unix, `keep` warns on stderr if a loaded config file is group/world-readable.

Check setup:

```bash
keep status
```

## Quick start

```bash
keep alerts list --limit 20
keep alerts get <fingerprint>
keep incidents list --format table
keep workflows list
keep providers installed
keep api GET /healthcheck
```

Filter JSON with `--jq` or pipe to `jq`. Use `--format table` for human-readable lists.

## Write access

State-changing commands (enrich, assign, dismiss, incident updates, workflow runs, mutating `keep api` calls) are **read-only by default**. Enable writes in one of these ways (highest precedence first):

1. **`KEEP_ACCESS=write`** — non-interactive opt-in for CI or sandboxes. The CLI prints a one-line notice on stderr the first time a write runs.
2. **`keep access write --confirm write`** — interactive; requires a TTY (recommended for humans).
3. **`"access": "write"` in `config.json`** — persistent non-interactive opt-in (use with care).

Return to read-only with `keep access read` or `KEEP_ACCESS=read`.

## Agent skill

```bash
keep skill print
keep skill install --agent cursor
```

## Why a CLI instead of MCP?

- **No server process** — agents invoke a binary; no MCP host wiring or long-lived connections.
- **Stable, pipe-friendly output** — JSON stdout and `--jq` fit existing agent toolchains.
- **Explicit safety** — read-only default, human-gated writes, and `untrusted` markers on alert content.
- **Escape hatch** — `keep api` covers the full OpenAPI surface without generating dozens of MCP tools.

See [docs/AGENT.md](docs/AGENT.md) for setup steps, command tables, and troubleshooting.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/keep
```

## License

MIT — see [LICENSE](LICENSE). Third-party notices in [NOTICE](NOTICE) and [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES).
