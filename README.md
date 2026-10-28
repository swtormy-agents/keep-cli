# keep-cli

Agent-friendly CLI for [Keep](https://github.com/keephq/keep) (AIOps). One static binary, JSON on stdout, no MCP server required.

## Install

```bash
go install github.com/swtormy-agents/keep-cli/cmd/keep@latest
```

Or build from source:

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

Override the config path with `KEEP_CONFIG`. A project-local `.keep/config.json` is also read.

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

State-changing commands (enrich, assign, dismiss, incident updates, workflow runs, mutating `keep api` calls) are **read-only by default**. The user must enable writes locally:

```bash
keep access write --confirm write
```

Agents should not run this; ask the user to do it in their own terminal. Return to read-only with `keep access read`.

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

MIT — see [LICENSE](LICENSE). Third-party notices in [NOTICE](NOTICE).
