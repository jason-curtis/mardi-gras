# Mardi Gras

Terminal UI for Beads issue tracking. Built with Go, Bubble Tea, and Lip Gloss.

## Local Installation

The compiled binary is symlinked into PATH:
- **Symlink**: `~/.local/bin/mg` → `<rig>/mayor/rig/mg`
- **Runtime**: Go 1.25+
- After rebuilding (`make build`), the symlink picks up the new binary automatically

## Development Commands

- `make build` — Build `mg` binary
- `make dev` — Build and run with sample data
- `make test` — Run tests
- `make fmt` — Format code
- `make lint` — Run golangci-lint

## Verification (Required)

Every code change MUST be verified end-to-end before completion:

1. `make build` — Rebuild the binary
2. Confirm `~/.local/bin/mg` resolves to the new build (symlink)
3. Open a tmux pane and run `mg` to visually verify the change works
   - Use `tmux split-window -h 'mg; read'` or similar to inspect output
   - Check that the UI renders correctly and the change is visible
