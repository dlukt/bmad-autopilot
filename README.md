# bmad-autopilot

Manual loop runner for BMAD sprint stories, implemented as a Go Cobra CLI.

## Run

```bash
go install github.com/dlukt/bmad-autopilot@latest
# cd to your project root
bmad-autopilot
```

Defaults:

- Status file: inferred from current working directory as `<cwd>/_bmad-output/implementation-artifacts/sprint-status.yaml`
- Brain: `deterministic`
- Workdir: inferred from the status file path (the project directory before `_bmad-output/`)
- Timeout: disabled
- Copilot model: unset
- Copilot execution: fresh SDK client/session per command, with `--yolo --no-ask-user -s`
- Slash commands:
  - create: `/bmad-create-story`
  - dev: `/bmad-dev-story`
  - review: `/bmad-code-review`
  - review options: `yolo and fix findings if any, or don't if not. If none are found git commit & push, only if none are found.`
- Logging: each action prints the raw Copilot output block plus a one-line summarized `RESULT` (enabled by default)
- Live command output: streamed from Copilot events when `--show-command-output=true`
- TUI: Bubble Tea interface is enabled by default on interactive terminals (`--tui=true`)

Interactive controls:

- `Ctrl+S`: request graceful stop after the current command finishes
- `Ctrl+A`: cancel a previously requested graceful stop
- `Ctrl+P`: toggle system poweroff at full loop completion (all non-retrospective stories done)
- `Ctrl+C`: cancel the active command
- `Up/Down`, `PgUp/PgDn`: scroll output in TUI mode
- `q`: quit TUI after the run has ended

## Useful flags

- `--status-file <path>`
- `--brain <glm-5|deterministic>`
- `--workdir <path>`
- `--copilot-model <model-id>`
- `--show-command-output <true|false>` (default: `true`)
- `--timeout <duration>` (use `0` to disable timeout)
- `--tui <true|false>` (default: `true`, only active on interactive terminals)
- `--create-story-slash-command </command>`
- `--dev-story-slash-command </command>`
- `--code-review-slash-command </command>`
- `--create-story-slash-options "<extra args>"`
- `--dev-story-slash-options "<extra args>"`
- `--code-review-slash-options "<extra args>"`
