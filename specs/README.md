# milbi specs

milbi is a small Go CLI that runs **restic** backups (local repos and Backblaze B2) and **rsync** syncs on the machine it is installed on. restic and rsync do the actual work. milbi only owns config, execution, state, and presentation.

Implement the specs in order. A spec is done when all of its acceptance criteria pass in CI.

| # | Spec | Covers |
|---|---|---|
| 01 | [language.md](01-language.md) | Go, tooling, host prerequisites |
| 02 | [release.md](02-release.md) | GitHub Actions CI and binary release |
| 03 | [testing.md](03-testing.md) | Test rules and coverage gate |
| 04 | [config.md](04-config.md) | YAML config and validation |
| 05 | [backup.md](05-backup.md) | restic backup jobs, state, exit codes |
| 06 | [sync.md](06-sync.md) | rsync sync jobs |
| 07 | [ui.md](07-ui.md) | `status` command and terminal UI |

**Out of scope for now:** OS scheduler integration (launchd/systemd), notifications, restore, keychain secrets, and any web UI.

**Rules for the agent:**
- Ask before adding a dependency that isn't listed in 01.
- Ask before changing anything that deletes data (`--delete`, `forget`, `prune`).
- Prefer simple, explicit code over abstraction.
