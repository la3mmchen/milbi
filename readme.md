# milbi

helps to organize backups. with a config. yaml!

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

as i work mostly with restic at the moment, milbis best user experience is with restic. but it should work with borg as well - in theory.

## features

### backup

at the moment, mibli can be configured to do backups with the following technologies:

- backups with [restic](https://restic.readthedocs.io/)
- backups with [borg](https://borgbackup.readthedocs.io/)

### synchronization

additionally, milbi can:

- copy directories with rsync
- sync to backblaze b2 with [b2 cli](https://www.backblaze.com/b2/docs/quick_command_line.html)

### verification

```bash
$ ./milbi.py check
doing my-repo
using temporary cache in /var/folders/vz/ct7cy2591p58dkbxflsgh18dgxl8xl/T/restic-check-cache-4190130004
create exclusive lock for repository
load indexes
check all packs
check snapshots, trees and blobs
[0:10] 100.00%  1 / 1 snapshots

no errors were found
```

in addition to running checks on the repositories, milbi can randomly fuzz through the files that should be backuped and search them in the backup repository.

```bash
/milbi.py --config ./tests/files/clitest-config.yaml fuzz --count 1 --verify

fuzzing around in configured backups..
preseeding files for ./tests/files/randomfiles. this might take a while..
...by checking for file file2.txt [repo [./tests/files/resticrepo] / target [./tests/files/randomfiles]]
restoring <Snapshot ac2c866a of [~/milbi/tests/files/randomfiles] at 2022-09-22 17:04:46.711727 +0200 CEST by user@test0r> to ~/milbi/./tests/files/resticrepo-restore/restic
verifying files in ~/milbi/./tests/files/resticrepo-restore/restic
finished verifying 1 files in ~/milbi/./tests/files/resticrepo-restore/restic (took 4ms)

```

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

## flows

milbi supports the invocation of multiple subcommands via the flows config.

an example can look like this. with this milbi first creates a backup, prunes, then checks the config, and afterwards executes the configured syncs.

```yaml
(...)
flows:
  - name: default
    tasks:
      - backup
      - prune
      - check
      - sync
```

start the flow via `flows` subcommand:

```
milbi --config tests/files/clitest-config.yaml flows --flow=default
(...)
flow [default]: backup > check > prune > sync
```

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

## development stuff

this repository contains a Taskfile that helps with all the different tasks:

```bash
$ task
task: Available tasks for this project:
* dev:lint:             run lint
* reqs:                 check requirements
* test:clitests:        run clitests
* test:prepare:         prepare for tests
* test:unittests:       run unittests
* tests:                run all tests
```