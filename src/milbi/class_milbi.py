#!/usr/bin/env python3

import os
import yaml
import sys
import shlex
import subprocess
import calendar
import time


class Milbi():
    """
    run backups from declarative config. intended to replace the random bash scripts I had on my disk.

    Parameters
    ----------
    config: string
        file of local config

    debug: bool
        print additional output

    """
    config = None
    debug = None

    _content = None
    _fileLocation = None
    _logfileHandle = None
    _timestamp = None
    _shellEnv = None

    def __init__(self, config="~/.milbi/config.yaml", debug=False):
        self.debug = debug

        self._timestamp = calendar.timegm(time.gmtime())

        # handle relative paths for config
        self._fileLocation = self._makePathAbsolute(config)

        # load content from config file
        self._content = self._loadConfig(self._makePathAbsolute(self._fileLocation))

        # handle relative paths
        self._content['global']['logfile'] = self._makePathAbsolute(self._content['global']['logfile'])
        self._content['global']['restore']['dir'] = self._makePathAbsolute(self._content['global']['restore']['dir'])

        # create a filehandle to log
        print(f"logging to {self._content['global']['logfile']}")
        self._logfileHandle = open(self._content['global']['logfile'], "a")

        if self.debug:
            self._toConsole("---")
            self._toConsole("loaded config: ")
            for item in vars(self).items():
                self._toConsole("`-  %s: %s" % item)
            self._toConsole("---")

        if self._content is None:
            self._toConsole("ERROR: no config file found.")
            sys.exit(1)

        self._shellEnv = os.environ.copy()

    def __del__(self):
        self._logfileHandle.close()

    def sync(self):
        """
        execute the configured syncs.

        syncs can be configured in the config with the `syncs` list.
        sync uses rsync that must be present on your local machine.

        example:

            syncs:
              - name: "sync-name"
                source: "~/local-folder"
                target: "~/local-target"

        Parameters
        ----------

        """
        if 'syncs' in self._content and len(self._content['syncs']) > 0:
            for syncconfig in self._content['syncs']:
                self._toConsole(f"doing sync of {syncconfig['name']} {syncconfig['source']} -> {syncconfig['target']}")
                # handle rsync
                if syncconfig['type'] == 'rsync' and os.path.exists(syncconfig['source']) and os.path.exists(syncconfig['target']):
                    try:
                        self._doRsync(syncconfig['source'], syncconfig['target'])
                    except Exception as e:
                        self._toConsole(f"ERROR: ({e}).")
                        sys.exit(1)
                # handle b2
                elif syncconfig['type'] == 'b2' and os.path.exists(syncconfig['source']):
                    try:
                        self._doB2Sync(syncconfig['source'], syncconfig['target'])
                    except Exception as e:
                        self._toConsole(f"ERROR: ({e}).")
                        sys.exit(1)
                else:
                    self._toConsole(f"`- config not valid at the moment. skipping")
                    pass
                self._toConsole("")

    def config(self):
        """
        shows the current config that is loaded from the configured file.

        Parameters
        ----------

        """
        if self._content is None:
            self._toConsole("ERROR no valid config found")
        else:
            self._toConsole(yaml.dump(self._content))

    def backup(self, simulate=False):
        """
        executes the configured backups.

        backups can be configured using borg or restic.
        both tools must be present on the local machine. for both the repository must be initialized outside
        of milbi (e.g. restic init --repo my-backup)

        example how to configure the restic backup:

          restic:
            enabled: True
            bin: "/usr/local/bin/restic"
            repos:
              - passphrase: "abcdefgh"
                repo: "~/backups-restic/"
                excludes: "*github.com*"
                keep: 3
                targets:
                  - ~/Documents

        Parameters
        ----------
        simulate: bool
            if true simulates a backup if this is possible
        """
        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            for item in self._content['restic']["repos"]:
                self._toConsole(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "backup",
                    "--verbose",
                    "--compression",
                    "auto",
                    "--host",
                    self._content['global']['hostalias'],
                    "--repo",
                    item['repo'],
                    "--exclude",
                    item['excludes'],
                ]
                cmd.extend(item['targets'])

                if simulate:
                    cmd.append("--dry-run")

                try:
                    self._cmdRunRestic(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

        if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
            for item in self._content['borgbackup']["repos"]:
                self._toConsole(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "create",
                    "--compression",
                    "zlib,6",
                    "--exclude",
                    item['excludes'],
                ]

                if simulate:
                    cmd.append("--dry-run")

                cmd.append(f"{item['repo']}::{self._timestamp}")
                cmd.extend(item['targets'])

                try:
                    self._cmdRunBorg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

    def check(self):
        """
        check the backup repos if the undelying tools support this operation.

        if borgbackup is used:
          - execute borg check on the last archive
          - run dry-run extract on the last archive

        if restic is used:
          - a restic check is executed on the repo

        for restic:
          - check the repo

        Parameters
        ----------

        """
        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            for item in self._content['restic']["repos"]:
                self._toConsole(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "check",
                    "--repo",
                    item['repo'],
                ]

                try:
                    self._cmdRunRestic(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

        if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
            for item in self._content['borgbackup']["repos"]:
                self._toConsole(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "check",
                    "--verify-data",
                    "--last",
                    "1",
                    self._content['borgbackup']['repo']
                ]

                try:
                    self._cmdRunBorg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

                # test if last backup can be extracted
                self._logfileHandle.write(f"{time.ctime(self._timestamp)} try to extract repo \n")

                cmd = [
                    "list",
                    "--short",
                    "--last",
                    "1",
                    f"{self._content['borgbackup']['repo']}"
                ]

                try:
                    _, stdout, _ = self._cmdRunBorg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

                lastArchive = stdout.rstrip()

                self._logfileHandle.write(f"{time.ctime(self._timestamp)} using archive {lastArchive}. \n")

                cmd = [
                    "extract",
                    "--dry-run",
                    "--list",
                    "--exclude",
                    self._content['borgbackup']['excludes'],
                    f"{self._content['borgbackup']['repo']}::{lastArchive}"
                ]

                try:
                    self._cmdRunBorg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

    def info(self):
        """
        show info of the existing repos (snapshots, repos, ...)

        Parameters
        ----------

        """
        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            for item in self._content['restic']['repos']:
                self._toConsole(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "snapshots",
                    "--repo",
                    item['repo'],
                ]

                try:
                    self._cmdRunRestic(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

        if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
            for item in self._content['borgbackup']["repos"]:
                self._toConsole(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "info",
                    item['repo']
                ]
                try:
                    self._cmdRunBorg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

                cmd = [
                    "list",
                    item['repo']
                ]
                try:
                    self._cmdRunBorg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

    def prune(self):
        """
        prune old snapshots within the configured repos if the tools support this operation.

        Parameters
        ----------

        """
        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            for item in self._content['restic']['repos']:
                self._toConsole(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "forget",
                    f"--keep-daily={item['keep']}",
                    "--repo",
                    item['repo']
                ]

                print(f"{cmd}")
                print(f"{shlex.join(cmd)}")
                try:
                    self._cmdRunRestic(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._toConsole(f"ERROR: ({e}).")
                    sys.exit(1)

        if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
            cmd = [
                "prune",
                "--verbose",
                "--list",
                f"--keep-daily={self._content['borgbackup']['keep']}",
                self._content['borgbackup']['repo']
            ]
            try:
                self._cmdRunBorg(cmd=cmd, passphrase=item['passphrase'])
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)

    def get(self, resticfilter=None, repo=None):
        """
        restore the repos into a local folder so files can be used.

        Parameters
        ----------
        resticfilter: string
            FOR RESTIC: only extract certain paths of backup (e.g. *ssh*)

        repo: string
            select the repository to extract via its name and not via an interactive selector.
        """
        if self._content['global']['restore']['dir']:
            if not os.path.exists(os.path.dirname(self._content['global']['restore']['dir'])):
                self._toConsole(f"ERROR: target directory {self._content['global']['restore']['dir']} does not exists.")

        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            repoToGet = None
            if repo is None:
                repoToGet = self._content['restic']['repos'][int(self._askRepoToRestore(self._content['restic']['repos']))]
            else:
                for item in self._content['restic']['repos']:
                    if repo.lower() in item['repo']:
                        repoToGet = item

            if repoToGet is None:
                self._toConsole("ERROR: Repo to extract not found.")
                sys.exit(1)

            self._toConsole(f"doing extraction of {repoToGet['repo']}")

            cmd = [
                "restore",
                "latest",
                "--repo",
                repoToGet['repo'],
                "--target",
                f"{self._content['global']['restore']['dir']}/restic"
            ]

            if resticfilter is not None:
                self._toConsole("HINT: filters are not quite fully integrated.")
                cmd.extend([
                    '--path',
                    resticfilter
                ])

            try:
                self._cmdRunRestic(cmd=cmd, passphrase=repoToGet['passphrase'])
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)

        if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
            cmd = [
                "extract",
                self._content['borgbackup']['repo'],
                f"{self._content['global']['restore']['dir']}/borgbackup"
            ]
            try:
                self._cmdRunBorg(cmd=cmd, passphrase=item['passphrase'])
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)

    def _cmdRunRestic(self, cmd, passphrase):
        """
        generic function to run restic commands.

        Parameters
        ----------
        cmd: string
            command to execute

        Returns
        -------
        result.returncode: int
            return code of command
        stdout: list
            outputs of command
        stderr: list
            error output of command
        """
        self._shellEnv['RESTIC_PASSWORD'] = passphrase

        command = [self._content['restic']['bin']]
        command.extend(cmd)

        if self.debug:
            print(f"{shlex.join(command)}")

        self._logfileHandle.write(f"{time.ctime(self._timestamp)} command: {shlex.join(command)} \n")

        try:
            result = subprocess.run(command, capture_output=True, env=self._shellEnv, encoding='utf-8')
            self._toConsole(result.stdout)

            if self.debug:
                self._toConsole(result.stderr)

            if result.returncode != 0:
                raise Exception(f"{command} not succesful. {result.stderr}")
        except Exception as e:
            raise Exception(e)

        # handle log output
        self._logfileHandle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
        self._logfileHandle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

        return (result.returncode, result.stdout, result.stderr)

    def _cmdRunBorg(self, cmd):
        """
        generic function to run borg commands.

        Parameters
        ----------
        cmd: string
            command to execute

        Returns
        -------
        result.returncode: int
            return code of command
        stdout: list
            outputs of command
        stderr: list
            error output of command
        """
        self._shellEnv['BORG_PASSPHRASE'] = passphrase

        command = [self._content['borgbackup']['bin']]
        command.extend(cmd)

        if self.debug:
            print(f"{shlex.join(command)}")

        self._logfileHandle.write(f"{time.ctime(self._timestamp)} command: {shlex.join(command)} \n")

        try:
            result = subprocess.run(command, capture_output=True, env=self._shellEnv, encoding='utf-8')
            self._toConsole(result.stdout)

            if self.debug:
                self._toConsole(result.stderr)

            if result.returncode != 0:
                raise Exception(f"{command} not succesful. {result.stderr}")
        except Exception as e:
            raise Exception(e)

        # handle log output
        self._logfileHandle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
        self._logfileHandle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

        return (result.returncode, result.stdout, result.stderr)

    def _doRsync(self, source, target):
        """
        generic function to execute rsync

        Parameters
        ----------
        source: string
            rsync source

        target: string
            rsync target

        Returns
        -------
        result.returncode: int
            return code of command
        stdout: list
            outputs of command
        stderr: list
            error output of command
        """

        command = [
            '/usr/local/bin/rsync',
            '--stats',
            '--progress',
            '--verbose',
            '--archive',
            '--compress',
            '--delete',
            source,
            target
        ]

        if self.debug:
            print(f"{shlex.join(command)}")

        self._logfileHandle.write(f"{time.ctime(self._timestamp)} rsync command: {shlex.join(command)} \n")

        try:
            result = subprocess.run(command, capture_output=True, env=self._shellEnv, encoding='utf-8')
            self._toConsole(result.stdout)

            if self.debug:
                self._toConsole(result.stderr)

            if result.returncode != 0:
                raise Exception(f"{command} not succesful. {result.stderr}")
        except Exception as e:
            raise Exception(e)

        # handle log output
        self._logfileHandle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
        self._logfileHandle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

    def _doB2Sync(self, source, target):
        """
        generic function to execute b2 sync

        Parameters
        ----------
        source: string
            rsync source

        target: string
            rsync target

        Returns
        -------
        result.returncode: int
            return code of command
        stdout: list
            outputs of command
        stderr: list
            error output of command
        """

        command = [
            self._content['b2']['binary'],
            'sync',
            '--keepDays',
            '7',
            source,
            target
        ]

        for item in self._content['b2']['env']:
            self._shellEnv[item['name']] = item['value']

        if self.debug:
            print(f"{shlex.join(command)}")

        self._logfileHandle.write(f"{time.ctime(self._timestamp)} b2 command: {shlex.join(command)} \n")

        try:
            result = subprocess.run(command, capture_output=True, env=self._shellEnv, encoding='utf-8')
            self._toConsole(result.stdout)

            if self.debug:
                self._toConsole(result.stderr)

            if result.returncode != 0:
                raise Exception(f"{command} not succesful. {result.stderr}")
        except Exception as e:
            raise Exception(e)

        # handle log output
        self._logfileHandle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
        self._logfileHandle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

        return (result.returncode, result.stdout, result.stderr)

    def _makePathAbsolute(self, inPath):
        """
        function to make sure a path is an absolute path
        """

        if '~' in inPath:
            inPath = os.path.expanduser(inPath)
        if not os.path.isabs(inPath):
            inPath = os.path.join(os.getcwd(), inPath)
        absPath = inPath

        return absPath

    def _toConsole(self, message):
        """
        common function to print to console.
        """
        print(f"{message}")
        return True

    def _loadConfig(self, fle):
        """
        load the config from file
        """
        ymlcnt = None
        try:
            with open(fle) as file:
                ymlcnt = yaml.safe_load(file)
                file.close()
        except Exception as e:
            self._toConsole(f"Error while parsing connfig ({fle}): {e}")
        return ymlcnt

    def _askRepoToRestore(self, lst):
        """
        get a list of repos and print a user facing selector.
        """
        selectedNumber = None
        self._toConsole("Please select a repository: ")

        while True:
            for i, v in enumerate(lst):
                print(f" [{i}] - {v['repo']}")
            answer = input(f"--> select a repo [0 to {len(lst)-1}] from the list above or [c]ancel ".lower())

            if answer == "c":
                sys.exit(1)
            if int(answer) >= 0 and int(answer) <= len(lst)-1:
                selectedNumber = answer
                break

        return selectedNumber