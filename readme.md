# milbi

helps to organize backups. with a config. yaml!

I wrote it for myself to get rid of my random backup bash scripts that i had before.

```yaml
apiVersion: milbi/v2
kind: repo
metadata:
  name: my-backup-repo
  hostalias: "local"
spec:
  passphrase: "iamarandompasswort"
  generations: 7
  directory: "./tests/files/resticrepo"
  excludes:
    - "*resticrepo*"
  content:
    - ./tests/files/backup-folder
```

```bash
[~] $ milbi snapshot
Create a snapshot.
open repository
lock repository
using parent snapshot 16f5c660
load index files
start scan on [~/./tests/files/backup-folder]
start backup on [~/./tests/files/backup-folder]
scan finished in 0.228s: 2 files, 1 B
(...)
modified  /Users/alexkoehler/, saved in 0.008s (0 B added, 0 B stored, 0 B metadata)
modified  /Users/, saved in 0.014s (0 B added, 0 B stored, 306 B metadata)

Files:           0 new,     0 changed,     2 unmodified
Dirs:            0 new,     2 changed,     8 unmodified
Data Blobs:      0 new
Tree Blobs:      2 new
Added to the repository: 753 B (608 B stored)

processed 2 files, 1 B in 0:00
snapshot 5e9a299a saved
```

Milbi works with [restic](https://restic.net/) at the moment.

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