# 06 – Sync (rsync, b2)

## Config

A sync job has a `from` (a local path), a `to`, and optionally `excludes` and `delete` (default `false`).
The `to` decides the tool (see 04):

- a local path or `host:path` → **rsync**;
- `b2://bucket/path` → the Backblaze **`b2`** CLI; plain files land in the bucket (no encryption by milbi,
  history only via the bucket's lifecycle settings). Needs `b2_key_id` and `b2_key` (secret refs).

The `from` is passed through **verbatim**, so a trailing `/` keeps rsync's meaning: `~/Pictures/` syncs the
folder's contents, while `~/Pictures` syncs the folder itself. The README documents this.

## Execution (`milbi run <job>`)

### rsync

- Command: `rsync -a --partial --stats [--delete] [--exclude ...] <from> <to>`. No `--human-readable`:
  it turns the `--stats` numbers into units; milbi formats the bytes itself.
- `--delete` is added **only** when `delete: true`.
- `excludes` are the sync's own, passed as `--exclude`; the `defaults.excludes` apply to backups only.
- With `--dry-run`, milbi adds `-n -v` to rsync and runs it, so the user sees which files would change.
  A dry run takes no lock and writes no state.
- milbi parses `--stats` to get the number of files transferred and the bytes sent (rsync 3.x
  `Number of regular files transferred` / `Total bytes sent`, openrsync `Number of files transferred` /
  `Total sent: N B`); a line it can't read leaves 0.
- **openrsync** (macOS `/usr/bin/rsync`) supports every flag above, so it gets the same command.
  `doctor` warns about it.
- A non-zero rsync exit fails the job; the status file holds rsync's exit code in `exitCode` and its
  stderr in `error`, the counts in `sync` (`filesTransferred`, `bytesSent`).
- `run --all` runs every backup, then every sync, each in config order. A missing rsync is exit 3,
  like a missing restic for backups.

### b2

- Command: `b2 sync [--delete] [--dry-run] [--exclude-regex ...] <from> b2://bucket/path`.
- `--delete` is added **only** when `delete: true`.
- Credentials go only into the child environment as `B2_APPLICATION_KEY_ID` / `B2_APPLICATION_KEY`,
  never into argv, logs or state; milbi never runs `b2 account authorize`.
- Every `b2` call also gets `B2_ACCOUNT_INFO` pointing at a fresh per-run temp file, removed
  afterwards (also on failure), so b2 never reads or leaves a login in `~/.b2_account_info`.
- `excludes` globs are translated to `--exclude-regex`; a glob that can't be translated is a config error.
- milbi reads the uploaded / deleted counts from `b2 sync`'s output where it reports them.
- `b2` is optional: if it is not on PATH, a `b2://` job fails before starting with a clear error
  (`failed`, exit code as in 05); rsync jobs and backups are unaffected.

### Both

- State, logs, locking, and exit codes behave exactly as in 05.

## doctor

- `doctor` reports `b2` (`b2 version`) as optional: missing is fine, unless the loaded config has a
  `b2://` sync, then it is a missing tool (exit code 3).

## Acceptance criteria

The rsync criteria are proven by the e2e scenarios `sync-local`, `sync-delete`, `sync-dry-run` and
`sync-failure` against real rsync; the b2 criteria by testscripts with a fake `b2`.

- [ ] A sync to a local temp dir copies the files, and the status file shows the transferred count.
- [ ] Without `delete: true`, files removed from the source stay in the target.
- [ ] With `delete: true`, files removed from the source are removed from the target.
- [ ] `--dry-run` changes nothing in the target and prints the planned changes.
- [ ] rsync exit codes other than 0 result in `failed`, with the rsync error message in the status file.
- [ ] A `b2://` job runs `b2 sync` (fake b2) with `--delete` only when `delete: true`, `--dry-run` passed
  through, and the keys only in the environment (checked by the fake).
- [ ] A `b2://` job with no `b2` on PATH fails with a clear message; `doctor` then exits 3, and exits 0
  without a `b2://` sync.
