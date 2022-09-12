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