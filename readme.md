# milbi

__work in progress__

a way to describe local backups - in a declarative way.
i wrote it for myself to get rid of my random backup bash scripts that i had before.

```bash
./milby.py
```

## features

at the moment, mibli can be configured to do backups with following technologies:

- backups with [restic](https://restic.readthedocs.io/)
- backups with [borg](https://borgbackup.readthedocs.io/)

additionally, milbi supports:

- copy local directories with rsync
- sync to backblaze b2 with [b2 cli](https://www.backblaze.com/b2/docs/quick_command_line.html)

## configure

milbi reads yaml config files.

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

## run it

### installation

milbi needs the used tools installed in the system.

in addition to python milbi needs PyYaml to parse the configuration.

```bash

$ pip3 install -r requierements.txt

```

### prepare repos

milbi does not handle repository creation of any kind. just make sure to create the repo for the tool you want to use before.

#### restic

initialize a restic repo and create a secret key if you want to have one.

```bash

restic init --repo <local path>

```

#### borbackup

to be described

## todos

- add some tests to milbi source code
- provide more docu
