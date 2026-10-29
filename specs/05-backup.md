# 05 – Backup creation (restic)

## Commands

- `milbi init <repository>` runs `restic init`. It refuses if the repo is already initialized (`restic cat config` succeeds).
- `milbi run <job>...` runs the named backup jobs in order. `milbi run --all` runs every backup job in config order. `--dry-run` prints the commands without running them. Naming a sync is a config error until spec 06.
- `milbi snapshots <job>` lists that job's snapshots in each of its repositories.
- `milbi prune <repository>` runs `restic prune`. It is never run automatically.

## Backup job behaviour

A job backs up into each repository in its `to`, one after the other; the job's status is the worst of them.

1. Resolve secrets. Pass them to restic **only via environment** (`RESTIC_PASSWORD`, `RESTIC_REPOSITORY`, and the repository's `env`). Inherited `RESTIC_PASSWORD*` and `RESTIC_REPOSITORY*` variables are dropped, so they can't override the config.
2. Run `restic backup --json --tag milbi --tag job:<name> [--exclude ...] <paths>`.
3. Parse the final JSON `summary` message to get files new/changed, bytes added, and the snapshot id. Progress (`status`) messages are not logged.
4. If `retention` is set and the backup succeeded (exit 0), run `restic forget --tag job:<name> --group-by host,tags --keep-*`. Scoping by the job tag is mandatory, so jobs sharing a repo never delete each other's snapshots. A failing `forget` fails the job.
5. restic exit code 3 (some files unreadable) is recorded as `warning`, not `failed`; retention is not applied after such an incomplete snapshot.
6. Only one run per job at a time, enforced by `flock` on `locks/<job>.lock` in the state dir (released by the kernel if milbi dies, so no stale locks). A busy lock means the job is `skipped`; a skipped run leaves the state file of the running one alone.

## State

The default state dir is `$XDG_STATE_HOME/milbi`, else `~/.local/state/milbi`.

- `status/<job>.json` is written as `running` at start, then atomically replaced with the result. It holds `status` (success, warning, failed), start and end timestamps, duration, `lastSuccessAt` (set by success and warning), the exit code, an error message, stats per repository, and the log file path.
- `logs/<job>/<timestamp>.log` holds the full child output. milbi keeps the last 30 logs per job.

## Exit codes (all `run` commands)

| Code | Meaning |
|---|---|
| 0 | All jobs succeeded or ended in warning |
| 1 | At least one job failed |
| 2 | Config error |
| 3 | Missing prerequisite |
| 4 | A job was skipped because of its lock (and none failed) |

## Acceptance criteria

Each is proven by an e2e scenario in `e2e/testdata/` (named in brackets).

- [x] `init` followed by `run documents` against a local repo creates a snapshot tagged `job:documents`. [backup-local]
- [x] A second job in the same repo with `retention: {last: 1}` doesn't remove the first job's snapshots. [backup-shared-repo]
- [x] The repository password never appears in argv, logs, or state files. [backup-local; argv in `internal/cli/testdata/script/backup.txtar`]
- [x] An unreadable file in the source yields `warning` and exit 0. [backup-unreadable]
- [x] Running the same job twice in parallel results in one `skipped` and exit 4. [backup-lock]
