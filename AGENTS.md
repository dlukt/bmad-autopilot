# Repository Guidelines

## Project Structure & Module Organization
`main.go` is the CLI entrypoint. `cmd/` contains Cobra command wiring, Bubble Tea TUI behavior, hotkey handling, and system poweroff integration. `internal/orchestrator/` owns sprint-status parsing, action planning, loop control, and Copilot execution. `internal/brain/` contains result summarizers such as `deterministic` and `glm-5`. Tests live beside the code as `*_test.go`; there is no separate fixtures or assets directory today.

## Build, Test, and Development Commands
- `go run . --help` shows global flags and available subcommands.
- `go run . run --status-file /path/to/sprint-status.yaml` runs the loop locally against a specific BMAD status file.
- `go build ./...` compiles all packages.
- `go test ./...` runs the full test suite.
- `go test ./... -cover` checks coverage for touched packages before a PR.
- `gofmt -w <changed-files>` formats modified Go files; run it before committing.

## Coding Style & Naming Conventions
Use standard Go formatting and import ordering; let `gofmt` define indentation and spacing. Keep package names short and lowercase. Exported identifiers use `CamelCase`; internal helpers use `camelCase`. Keep CLI-specific concerns in `cmd/` and move reusable logic into `internal/`. Prefer explicit configuration structs, as seen with `rootOptions`, `orchestrator.Config`, and `SDKExecutorOptions`.

## Testing Guidelines
Write tests next to the implementation and name them `TestXxxBehavior`. Favor table-driven tests for status/action matrices and edge-case flows. There is no enforced coverage gate, but changes should preserve or improve coverage in the affected package. Re-run `go test ./...` after changes to loop control, git state checks, or TUI/hotkey behavior.

## Commit & Pull Request Guidelines
Recent history uses short, imperative commit subjects, sometimes with conventional prefixes like `feat:` or `chore:`. Follow that pattern, for example: `Fix hotkey terminal mode` or `feat: add optional output logging`. PRs should summarize behavior changes, link the relevant issue or story, and list verification steps. Include terminal output or screenshots when changing TUI or hotkey interactions.

## Security & Configuration Tips
Do not commit `.env`, coverage files, or local binaries. `glm-5` summarization requires `ZAI_API_KEY`; deterministic mode is the default fallback. End-to-end runs also assume GitHub Copilot CLI auth, a valid `_bmad-output/implementation-artifacts/sprint-status.yaml`, and caution around `Ctrl+P`, which can trigger `sudo systemctl poweroff` after a full loop completes.
