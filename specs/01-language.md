# 01 – Language and tooling

## Behaviour

- milbi is written in **Go** (current stable release) and builds with `CGO_ENABLED=0` into a single static binary.
- Module path: `github.com/la3mmchen/milbi`.
- Allowed dependencies: `spf13/cobra` (CLI), `charmbracelet/bubbletea`, `bubbles` and `lipgloss` (UI), `go.yaml.in/yaml/v3` (config), `rogpeppe/go-internal/testscript` (CLI tests). Logging uses the standard library's `log/slog`.
- restic and rsync are **never bundled**. milbi looks them up on `PATH`, unless the config sets an explicit binary path.
- A `Taskfile.yaml` provides `task build`, `task test`, `task lint`, and `task test:integration`.

## Layout

```
cmd/milbi/          main.go only
internal/cli/       cobra commands, thin
internal/config/    loading, validation
internal/restic/    command building, output parsing
internal/rsync/     command building, output parsing
internal/runner/    job execution, exit codes
internal/state/     status and log files
internal/tui/       terminal UI
```

## Commands

- `milbi version` prints the version, commit, and build date, all injected via ldflags.
- `milbi doctor` checks that restic and rsync are present and prints their versions. It warns if `rsync` is openrsync (the macOS default). It exits with code 3 if a required binary is missing.

## Acceptance criteria

- [X] `task build` produces a working binary for the host OS/arch.
- [X] `milbi version` shows non-empty version, commit, and date in release builds.
- [X] `milbi doctor` exits 0 with both tools present, and 3 with restic removed from `PATH`.
