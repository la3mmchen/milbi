# 06 – Sync (rsync)

## Config

A sync job has a `source`, a `destination` of type `rsync` with a `target` (a local path or an rsync remote such as `host:/path`), and optionally `excludes` and `delete` (default `false`).

The `source` is passed through **verbatim**, so a trailing `/` keeps rsync's meaning: `~/Pictures/` syncs the folder's contents, while `~/Pictures` syncs the folder itself. The README documents this.

## Execution (`milbi run <job>`)

- Command: `rsync -a --partial --human-readable --stats [--delete] [--exclude ...] <source> <target>`.
- `--delete` is added **only** when `delete: true`.
- With `--dry-run`, milbi adds `-n` to rsync and runs it, so the user sees what would change.
- milbi parses `--stats` to get the number of files transferred and the bytes sent.
- **openrsync** (detected from `rsync --version`) gets only the flags it supports. `doctor` warns about it.
- State, logs, locking, and exit codes behave exactly as in 05.

## Acceptance criteria

- [ ] A sync to a local temp dir copies the files, and the status file shows the transferred count.
- [ ] Without `delete: true`, files removed from the source stay in the target.
- [ ] With `delete: true`, files removed from the source are removed from the target.
- [ ] `--dry-run` changes nothing in the target and prints the planned changes.
- [ ] rsync exit codes other than 0 result in `failed`, with the rsync error message in the status file.
