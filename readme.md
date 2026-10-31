# milbi

milbi runs [restic](https://restic.net/) backups and rsync syncs on the machine it is installed on, driven by one YAML config. restic and rsync do the actual work; milbi owns config, execution, state and presentation.

This is a rewrite from scratch. The design lives in [specs/](specs/README.md) and is implemented spec by spec; a spec is done when its acceptance criteria pass in CI. The previous implementation (config `apiVersion: milbi/v2`) is available under the `2.0.x` tags.

## Status

Implemented so far:

- `milbi version` prints version, commit and build date.
- `milbi doctor` checks that `restic` and `rsync` are on `PATH`, prints their versions and warns when `rsync` is openrsync (the macOS default). Exit code 3 means a required binary is missing.

The config format, backup and sync jobs, state files and the terminal UI are specified in [specs/04](specs/04-config.md) to [specs/07](specs/07-ui.md) and not implemented yet. [example-config.yaml](example-config.yaml) shows the target format.

## Requirements

- restic and rsync 3.x on `PATH`. They are never bundled. On macOS install rsync with Homebrew; the bundled openrsync lacks options milbi needs.
- Go (version in `go.mod`) and [Task](https://taskfile.dev) to build from source.

## Development

```bash
task build             # build bin/milbi with version info from git
task test              # unit tests with per-package coverage gate (85%)
task test:e2e          # end-to-end scenarios against real restic and rsync, skipped when missing
task lint              # gofmt and go vet
task lint:ci           # golangci-lint, same version as CI
```

Releases are built by GoReleaser from tags matching `v*.*.*`; see [specs/02](specs/02-release.md).
