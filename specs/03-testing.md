# 03 – Testing

## Behaviour

- **Every function has at least one test.** Functions with branches use table-driven tests that cover each branch, including the error paths.
- **Coverage gate:** at least 85% statement coverage per package, excluding `cmd/milbi`. CI fails below this.
- **Unit tests never run real restic or rsync.** Instead, testscript puts fake `restic` and `rsync` scripts on `PATH` that record their argv and environment to a file and print fixture output.
- Fixtures for restic JSON output and rsync `--stats` output are recorded from the real tools and stored in `testdata/`.
- **Tests never touch the real home directory or network.** `HOME` and `XDG_*` point to temp dirs.
- **Integration tests** (build tag `integration`) use a real local restic repo and real rsync in a temp dir, covering init, backup, snapshots, and sync. They skip cleanly when a tool is missing.

## What must be asserted

- The exact argv passed to restic and rsync for every job type and option.
- Secrets appear only in the child environment, **never in argv, logs, or state files**.
- Exit codes for success, warning, failure, and config error.
- Config validation reports every error case listed in 04.

## Acceptance criteria

- [X] `task test` passes and prints coverage per package.
- [X] CI fails if any package drops below 85%.
- [X] `task test:integration` passes on a machine with restic and rsync installed.
