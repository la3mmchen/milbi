# 05 – Backup creation (restic)

## Commands

- `milbi init <destination>` runs `restic init`. It refuses if the repo is already initialized.
- `milbi run <job>...` runs the named jobs in order. `milbi run --all` runs every job in config order. `--dry-run` prints the commands without running them.
- `milbi snapshots <job>` lists that job's snapshots.
- `milbi prune <destination>` runs `restic prune`. It is never run automatically.

## Backup job behaviour

1. Resolve secrets. Pass them to restic **only via environment** (`RESTIC_PASSWORD`, `RESTIC_REPOSITORY`, and the destination's `env`).
2. Run `restic backup --json --tag milbi --tag job:<name> [--exclude ...] <paths>`.
3. Parse the final JSON `summary` message to get files new/changed, bytes added, and the snapshot id.
4. If `retention` is set and the backup succeeded, run `restic forget --tag job:<name> --group-by host,tags --keep-*`. Scoping by the job tag is mandatory, so jobs sharing a repo never delete each other's snapshots.
5. restic exit code 3 (some files unreadable) is recorded as `warning`, not `failed`.
6. Only one run per job at a time, enforced by a lock file in the state dir. A busy lock means the job is `skipped`.

## State

The default state dir is `~/.local/state/milbi`.

- `status/<job>.json` is written as `running` at start, then atomically replaced with the result. It holds `status` (success, warning, failed, skipped), start and end timestamps, duration, `lastSuccessAt`, the exit code, an error message, stats, and the log file path.
- `logs/<job>/<timestamp>.log` holds the full child output. milbi keeps the last 30 logs per job.

## Exit codes (all `run` commands)

| Code | Meaning |
|---|---|
| 0 | All jobs succeeded or ended in warning |
| 1 | At least one job failed |
| 2 | Config error |
| 3 | Missing prerequisite |
| 4 | A job was skipped because of its lock |

## Acceptance criteria

- [ ] `init` followed by `run documents` against a local repo creates a snapshot tagged `job:documents`.
- [ ] A second job in the same repo with `retention: {last: 1}` doesn't remove the first job's snapshots.
- [ ] The repository password never appears in argv, logs, or state files.
- [ ] An unreadable file in the source yields `warning` and exit 0.
- [ ] Running the same job twice in parallel results in one `skipped` and exit 4.
