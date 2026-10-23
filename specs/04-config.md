# 04 – YAML config

## Behaviour

- The default path is `~/.config/milbi/config.yaml`, overridable with `--config` or `MILBI_CONFIG`.
- Decoding is strict: unknown fields are errors.
- `~` and `$VARS` in local paths are expanded. Relative paths resolve against the config file's directory.
- **Repositories** are shared: they are defined once and several backups reference them by name in `to`.
  A sync's `to` is an inline target, not a reference: a local path or `host:path` (rsync), or
  `b2://bucket/path` (Backblaze, plain files via the `b2` CLI, see 06).
- Retention: a backup's `retention` replaces the default one; its `excludes` are added on top of the defaults.

```yaml
version: 1

defaults:
  excludes: ["**/node_modules", "**/.terraform", "**/.DS_Store"]
  retention: { last: 3, daily: 7, weekly: 4, monthly: 12 }

repositories:
  local:
    url: /Volumes/Backup/restic
    password: file:~/.config/milbi/secrets/local
  b2:
    url: s3:https://s3.eu-central-003.backblazeb2.com/my-bucket/laptop
    password: file:~/.config/milbi/secrets/b2-repo
    env:
      AWS_ACCESS_KEY_ID: env:B2_KEY_ID
      AWS_SECRET_ACCESS_KEY: file:~/.config/milbi/secrets/b2-app-key

backups:
  documents:
    from: [~/Documents, ~/Projects]
    to: [local, b2]
    excludes: ["**/build"]            # added on top of defaults

  configs:
    from: [~/.config, ~/.ssh]
    to: [b2]
    retention: { daily: 30, monthly: 24 }   # replaces the defaults

syncs:
  photos:
    from: ~/Pictures/
    to: nas.local:/volume1/mirror/photos/
    delete: false

  photos-b2:
    from: ~/Pictures/
    to: b2://my-photos/laptop
    b2_key_id: env:B2_KEY_ID
    b2_key: file:~/.config/milbi/secrets/b2-app-key
```

## Secrets

`password`, every `env` value and a sync's `b2_key_id` / `b2_key` must be a reference:

- `file:<path>` reads the file and trims the trailing newline. milbi warns if the file mode is broader than 0600.
- `env:<VAR>` reads the variable from milbi's own environment.

A plain literal value is a config error.

## Paths

- A sync's `from` keeps its trailing slash exactly as written (rsync copies the directory's contents
  with it, the directory itself without); path cleaning must not strip it.
- A sync `to` of the form `host:path` is remote: it is never `~`-expanded or resolved against the
  config directory.
- A sync `to` starting with `b2://` is a Backblaze target and is left unchanged. Any other URL-like
  target (`s3:`, `rclone:`, `<scheme>://`) is not supported yet; a future backend allows its own prefix.

## Validation (`milbi config validate`)

These are errors:

- `version` is missing or unsupported.
- A repository, backup or sync name doesn't match `[a-z0-9-]+`.
- A name is used by both a backup and a sync (`milbi run <name>` addresses both).
- A backup's `to` names a repository that doesn't exist, is empty, or lists one twice.
- A backup has no `from`, or a sync has no `from` or no `to`.
- A retention key is not one of `last`, `hourly`, `daily`, `weekly`, `monthly`, `yearly`, or its value is negative.
- A backup's effective retention (its own, else the default) keeps nothing: every value is zero or none is set
  (`forget` would remove every snapshot).
- An `env` key is not a valid variable name, or sets `RESTIC_PASSWORD*` (the password comes from `password`).
- A secret is an empty `file:` or `env:` reference. Any other value without one of these prefixes is a plain-text literal and is allowed.
- A sync `to` is URL-like but not `b2://` ("not supported yet"), or a `b2://` target has no bucket.
- A `b2://` sync lacks `b2_key_id` or `b2_key`, or a non-`b2://` sync sets either of them.

A local `from` path that doesn't exist is a warning, not an error (external volumes may be unmounted).

All errors are reported at once, with the line number where available. On success, the command prints
the backups with their repositories and the syncs with their targets.

## Acceptance criteria

- [ ] The example above validates, with exit code 0.
- [ ] Each error case above produces a clear message and exit code 2.
- [ ] An unknown field such as `pathz:` is rejected with its line number.
- [ ] A sync `from` with a trailing slash keeps it after loading; a `host:path` or `b2://` target is left unchanged.
- [ ] `to: s3:https://…` on a sync is rejected with exit code 2; a `b2://` sync without `b2_key` too.
