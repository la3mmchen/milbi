# 03 – Testing

## Behaviour

- **Every function has at least one test.** Functions with branches use table-driven tests that cover each branch, including the error paths.
- **Coverage gate:** at least 85% statement coverage per package, excluding `cmd/milbi`. CI fails below this.
- **Unit tests never run real restic or rsync.** Instead, testscript puts fake `restic` and `rsync` scripts on `PATH` that record their argv and environment to a file and print fixture output.
- Fixtures for restic JSON output and rsync `--stats` output are recorded from the real tools and stored in `testdata/`.
- **Tests never touch the real home directory or network.** `HOME` and `XDG_*` point to temp dirs.
- **Integration tests** (build tag `integration`) use a real local restic repo and real rsync in a temp dir, covering init, backup, snapshots, and sync. They skip cleanly when a tool is missing.

## End-to-end

- **End-to-end scenarios** (build tag `e2e`, `task test:e2e`) run the built `milbi` binary against real restic and rsync, as a user would. They skip cleanly when a tool is missing.
- One folder per scenario in `e2e/testdata/<scenario>/`: `script.txtar` (testscript: commands and expectations) plus its fixtures (`milbi.yaml`, a `files/` source tree), copied into the script's `$WORK`. `HOME` is `$WORK/home`. Folders starting with `_` are shared fixtures, not scenarios.
- A scenario asserts exit codes and output, then the real outcome: repository contents (restic called directly, for assertions only), the rsync target tree, and the state files under `HOME`.
- Specs 05–07 each add scenarios that prove their acceptance criteria; b2 scenarios skip unless `MILBI_E2E_B2_*` env vars are set.

## What must be asserted

- The exact argv passed to restic and rsync for every job type and option.
- Secrets appear only in the child environment, **never in argv, logs, or state files**.
- Exit codes for success, warning, failure, and config error.
- Config validation reports every error case listed in 04.

## Acceptance criteria

- [X] `task test` passes and prints coverage per package.
- [X] CI fails if any package drops below 85%.
- [X] `task test:integration` passes on a machine with restic and rsync installed.
- [ ] `task test:e2e` passes on a machine with restic and rsync installed and runs every scenario folder.
