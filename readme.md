# milbi

helps to organize backups. with a declarative config. yaml!

i wrote it for myself to get rid of my random backup bash scripts that i had before.

```bash
[~] $ milbi backup
logging to ~/.milbi/milbibackup.log
doing my-macbook-backup
open repository
lock repository
using parent snapshot e7000ce5
load index files
start scan on [~/repos ~/docs ~/.ssh]
start backup on [~/repos ~/docs ~/.ssh]
scan finished in 6.598s: 82805 files, 6.863 GiB

Files:          44 new,   134 changed, 82627 unmodified
Dirs:           12 new,   156 changed, 21316 unmodified
Data Blobs:    130 new
Tree Blobs:    162 new
Added to the repository: 7.458 MiB (3.232 MiB stored)

processed 82805 files, 6.863 GiB in 0:18
snapshot cece5d02 saved
```

## features

at the moment, mibli can be configured to do backups with the following technologies:

- backups with [restic](https://restic.readthedocs.io/)
- backups with [borg](https://borgbackup.readthedocs.io/)

additionally, milbi can:

- copy directories with rsync
- sync to backblaze b2 with [b2 cli](https://www.backblaze.com/b2/docs/quick_command_line.html)

## configuration

milbi understands yaml. run `milbi config --explain` to get a config explanation.


```yaml
---
borgbackup:
  bin: Path to borg binary to use.
  enabled: 'A trigger to activate the backup with borgbackup. (Options: True / False).'
  repos:
  - excludes: A pattern for files to be excluded from the repository; e.g. *github*
    keep: 'For prune actions: How many days to keep in the repository; e.g. 2 for 2 days'
    passphrase: Passphrase for the repository
    repo: Path to the existing borg repository.
(...)
```

see [config example](https://github.com/la3mmchen/milbi/blob/main/example-config.yaml) for a full example.

## run it

you can find a pypi package at https://pypi.org/project/milbi/

```bash
$ pip3 install milbi
$ milbi --help
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
