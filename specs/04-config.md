# 04 – YAML config

## Behaviour

- The default path is `~/.config/milbi/config.yaml`, overridable with `--config` or `MILBI_CONFIG`.
- Decoding is strict: unknown fields are errors.
- `~` and `$VARS` in paths are expanded. Relative paths resolve against the config file's directory.
- **Destinations** (where data goes) and **jobs** (what runs) are separate, so several jobs can share one destination.

```yaml
version: 1

destinations:
  b2:
    type: restic
    repository: s3:https://s3.eu-central-003.backblazeb2.com/my-bucket/laptop
    password: file:~/.config/milbi/secrets/b2-repo
    env:
      AWS_ACCESS_KEY_ID: env:B2_KEY_ID
      AWS_SECRET_ACCESS_KEY: file:~/.config/milbi/secrets/b2-app-key
  nas:
    type: rsync
    target: nas.local:/volume1/mirror/

jobs:
  documents:
    kind: backup
    destination: b2
    paths: [~/Documents, ~/Projects]
    excludes: ["**/node_modules"]
    retention: { last: 3, daily: 7, weekly: 4, monthly: 12 }
  photos:
    kind: sync
    destination: nas
    source: ~/Pictures/
    delete: false
```

## Secrets

`password` and every `env` value must be a reference:

- `file:<path>` reads the file and trims the trailing newline. milbi warns if the file mode is broader than 0600.
- `env:<VAR>` reads the variable from milbi's own environment.

A plain literal value is a config error.

## Validation (`milbi config validate`)

These are errors:

- `version` is missing or unsupported.
- A job or destination name doesn't match `[a-z0-9-]+`.
- A job references a destination that doesn't exist.
- A `backup` job points to a non-restic destination, or a `sync` job to a non-rsync destination.
- A backup job has no `paths`, or a sync job has no `source`.
- A secret uses a literal value or an unknown scheme.

All errors are reported at once, with the line number where available. On success, the command prints the jobs and their destinations.

## Acceptance criteria

- [ ] The example above validates, with exit code 0.
- [ ] Each error case above produces a clear message and exit code 2.
- [ ] An unknown field such as `pathz:` is rejected with its line number.
