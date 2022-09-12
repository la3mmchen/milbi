# milbi

handles local backup & to some point supports the restore process.

for me it is at least better then the random bash scripts i had before.

at the moment mibli can be configured to do backups with following technologies:

- backups with restic (https://restic.readthedocs.io/)
- backups with borg (https://borgbackup.readthedocs.io/)

additional milbi supports:

- copy with rsync

## help

```bash
NAME
    milbi - handles local backup & to some point supports the restore process.

SYNOPSIS
    milbi - COMMAND | VALUE

DESCRIPTION
    for me it is at least better then the random bash scripts i had before.

COMMANDS
    COMMAND is one of the following:

     backup
       Executes a backup

     check
       check backups if possible

     close
       close open backup

     config
       Shows the current config.

     get
       open backup (either by mounting or restoring)

     info
       show info of the existing repos (snapshots, repos, ...)

     prune
       if possible prune backups to

     sync
       execute configured syncs
```

## configure

milbi reads config files.

this is an example config for milbi:

```yaml
---
# set global options
global:
  debug: True
  logfile: "~/.milbi/milbibackup.log"
  restore:
      dir: "~/mnt"

# configure borg
borgbackup:
  enabled: False
  bin: "/usr/local/bin/borg"
  passphrase: "abcdefgh"
  keep: 7
  repo: "~/backups/"
  excludes: "*github.com*"
  targets:
    - ~/repos
    - ~/Documents

# configure restic
restic:
  enabled: True
  bin: "/usr/local/bin/restic"
  passphrase: "abcdefgh"
  repo: "~/backups-restic/"
  excludes: "*github.com*"
  keep: 7
  targets:
    - ~/repos
    - ~/Documents
    - ~/.ssh
    - ~/.zshrc
    - ~/.kube

# have some syncs
syncs:
  - name: "kdbx"
    source: "~/kdbx"
    target: "/Volumes/backup"
  - name: "backup"
    source: "~/backup"
    target: "/Volumes/backup"
```

## todo

- add some tests to milbi source code
- produce more help