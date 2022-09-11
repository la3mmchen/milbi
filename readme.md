# milbi

handles local backup & to some point supports the restore process.

for me it is at least better then the random bash scripts i had before.

at the moment mibli supports:

- backups with borg

## configure

milbi reads config files.

### borgbackup

this is how to configure milbi to do backups with borg:

```yaml

global:
  debug: True
  logfile: "~/borgbackup.log"
  restore:
      dir: "~/mnt"

borgbackup:
  bin: "<path to binary>"
  passphrase: "<key for decryption>"
  keep: <how many days before pruning archives>
  repo: "<local folder for borg repo>"
  excludes: "<some excludes in the paths, e.g. *github.com*>"
  targets:
    - <list of local directories to backup>
```
