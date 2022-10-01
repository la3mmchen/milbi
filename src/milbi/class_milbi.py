#!/usr/bin/env python3

import os
import yaml
import sys
import shlex
import random
import subprocess
import calendar
import time
from glob import glob
from hashlib import sha512

from .class_config import Config


class Milbi():
    """
    helps to organize backups. with a declarative config. yaml!
    intended to replace the random bash scripts I had on my disk.

    Parameters
    ----------
    config: string
        file of local config

    debug: bool
        print additional output

    """
    config = None
    debug = None

    _logfile_handle = None
    _timestamp = None
    _shell_env = None

    def __init__(self, config="~/.milbi/config.yaml", debug=False):
        self.debug = debug

        self._timestamp = calendar.timegm(time.gmtime())

        # load content from config file
        Config(config)

        # create a filehandle to log
        self._to_console(f"logging to {Config._CONFIG['global']['logfile']}")
        self._logfile_handle = open(Config._CONFIG['global']['logfile'], "a")

        if self.debug:
            self._to_console("---")
            self._to_console("loaded config: ")
            for item in vars(self).items():
                self._to_console("`-  %s: %s" % item)
            self._to_console("---")
        self._shell_env = os.environ.copy()

    def __del__(self):
        self._logfile_handle.close()

    def sync(self):
        """
        execute the configured syncs.

        syncs can be configured in the config with the `syncs` list.
        sync uses rsync that must be present on your local machine.

        example:

            syncs:
              - name: "sync-name"
                type: "rsync"
                source: "~/local-folder"
                target: "~/local-target"

        Parameters
        ----------

        """
        if 'syncs' in Config._CONFIG and len(Config._CONFIG['syncs']) > 0:
            for syncconfig in Config._CONFIG['syncs']:
                self._to_console(f"doing sync of {syncconfig['name']} {syncconfig['source']} -> {syncconfig['target']}")
                # handle rsync
                if syncconfig['type'] == 'rsync' and os.path.exists(syncconfig['source']) and os.path.exists(syncconfig['target']):
                    try:
                        self._do_rsync(syncconfig['source'], syncconfig['target'])
                    except Exception as e:
                        self._to_console(f"ERROR: ({e}).")
                        sys.exit(1)
                # handle b2
                elif syncconfig['type'] == 'b2' and os.path.exists(syncconfig['source']):
                    try:
                        self._do_b2_sync(syncconfig['source'], syncconfig['target'])
                    except Exception as e:
                        self._to_console(f"ERROR: ({e}).")
                        sys.exit(1)
                else:
                    self._to_console("`- config not valid at the moment. skipping")
                    pass
                self._to_console("")

    def config(self, explain=False):
        """
        shows the current config that is loaded from the configured file.

        Parameters
        ----------
        explain: bool
          if true print an example configuration.
        """
        if explain:
            self._to_console("---")
            self._to_console(f"{yaml.safe_dump(Config._ATTRIBUTES, width=1000)}")
        else:
            self._to_console(yaml.safe_dump(Config._CONFIG, indent=4))

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
        if 'restic' in Config._CONFIG and len(Config._CONFIG['restic'].keys()) > 0 and Config._CONFIG['restic']["enabled"]:
            for item in Config._CONFIG['restic']["repos"]:
                self._to_console(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "backup",
                    "--verbose",
                    "--compression",
                    "auto",
                    "--host",
                    Config._CONFIG['global']['hostalias'],
                    "--repo",
                    item['repo'],
                    "--exclude",
                    item['excludes'],
                ]
                cmd.extend(item['targets'])

                if simulate:
                    cmd.append("--dry-run")

                try:
                    self._cmd_run_restic(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

        if 'borgbackup' in Config._CONFIG and len(Config._CONFIG['borgbackup'].keys()) > 0 and Config._CONFIG['borgbackup']["enabled"]:
            for item in Config._CONFIG['borgbackup']["repos"]:
                self._to_console(f"doing {os.path.basename(item['repo'])}")
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
                    self._cmd_run_borg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
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
        if 'restic' in Config._CONFIG and len(Config._CONFIG['restic'].keys()) > 0 and Config._CONFIG['restic']["enabled"]:
            for item in Config._CONFIG['restic']["repos"]:
                self._to_console(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "check",
                    "--repo",
                    item['repo'],
                ]

                try:
                    self._cmd_run_restic(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

        if 'borgbackup' in Config._CONFIG and len(Config._CONFIG['borgbackup'].keys()) > 0 and Config._CONFIG['borgbackup']["enabled"]:
            for item in Config._CONFIG['borgbackup']["repos"]:
                self._to_console(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "check",
                    "--verify-data",
                    "--last",
                    "1",
                    Config._CONFIG['borgbackup']['repo']
                ]

                try:
                    self._cmd_run_borg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

                # test if last backup can be extracted
                self._logfile_handle.write(f"{time.ctime(self._timestamp)} try to extract repo \n")

                cmd = [
                    "list",
                    "--short",
                    "--last",
                    "1",
                    f"{Config._CONFIG['borgbackup']['repo']}"
                ]

                try:
                    _, stdout, _ = self._cmd_run_borg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

                lastArchive = stdout.rstrip()

                self._logfile_handle.write(f"{time.ctime(self._timestamp)} using archive {lastArchive}. \n")

                cmd = [
                    "extract",
                    "--dry-run",
                    "--list",
                    "--exclude",
                    Config._CONFIG['borgbackup']['excludes'],
                    f"{Config._CONFIG['borgbackup']['repo']}::{lastArchive}"
                ]

                try:
                    self._cmd_run_borg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

    def info(self):
        """
        show info of the existing repos (snapshots, repos, ...)

        Parameters
        ----------

        """
        if 'restic' in Config._CONFIG and len(Config._CONFIG['restic'].keys()) > 0 and Config._CONFIG['restic']["enabled"]:
            for item in Config._CONFIG['restic']['repos']:
                self._to_console(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "snapshots",
                    "--repo",
                    item['repo'],
                ]

                try:
                    self._cmd_run_restic(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

        if 'borgbackup' in Config._CONFIG and len(Config._CONFIG['borgbackup'].keys()) > 0 and Config._CONFIG['borgbackup']["enabled"]:
            for item in Config._CONFIG['borgbackup']["repos"]:
                self._to_console(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "info",
                    item['repo']
                ]
                try:
                    self._cmd_run_borg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

                cmd = [
                    "list",
                    item['repo']
                ]
                try:
                    self._cmd_run_borg(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

    def prune(self):
        """
        prune old snapshots within the configured repos if the tools support this operation.

        Parameters
        ----------

        """
        if 'restic' in Config._CONFIG and len(Config._CONFIG['restic'].keys()) > 0 and Config._CONFIG['restic']["enabled"]:
            for item in Config._CONFIG['restic']['repos']:
                self._to_console(f"doing {os.path.basename(item['repo'])}")
                cmd = [
                    "forget",
                    f"--keep-daily={item['keep']}",
                    "--repo",
                    item['repo']
                ]

                try:
                    self._cmd_run_restic(cmd=cmd, passphrase=item['passphrase'])
                except Exception as e:
                    self._to_console(f"ERROR: ({e}).")
                    sys.exit(1)

        if 'borgbackup' in Config._CONFIG and len(Config._CONFIG['borgbackup'].keys()) > 0 and Config._CONFIG['borgbackup']["enabled"]:
            cmd = [
                "prune",
                "--verbose",
                "--list",
                f"--keep-daily={Config._CONFIG['borgbackup']['keep']}",
                Config._CONFIG['borgbackup']['repo']
            ]
            try:
                self._cmd_run_borg(cmd=cmd, passphrase=item['passphrase'])
            except Exception as e:
                self._to_console(f"ERROR: ({e}).")
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
        if Config._CONFIG['global']['restore']['dir']:
            if not os.path.exists(os.path.dirname(Config._CONFIG['global']['restore']['dir'])):
                self._to_console(f"ERROR: target directory {Config._CONFIG['global']['restore']['dir']} does not exists.")

        if 'restic' in Config._CONFIG and len(Config._CONFIG['restic'].keys()) > 0 and Config._CONFIG['restic']["enabled"]:
            repoToGet = None
            if repo is None:
                repoToGet = Config._CONFIG['restic']['repos'][int(self._ask_repo_to_restore(Config._CONFIG['restic']['repos']))]
            else:
                for item in Config._CONFIG['restic']['repos']:
                    if repo.lower() in item['repo']:
                        repoToGet = item

            if repoToGet is None:
                self._to_console("ERROR: Repo to extract not found.")
                sys.exit(1)

            self._to_console(f"doing extraction of {repoToGet['repo']}")

            cmd = [
                "restore",
                "latest",
                "--repo",
                repoToGet['repo'],
                "--target",
                f"{Config._CONFIG['global']['restore']['dir']}/restic"
            ]

            if resticfilter is not None:
                self._to_console("HINT: filters are not quite fully integrated.")
                cmd.extend([
                    '--path',
                    resticfilter
                ])

            try:
                self._cmd_run_restic(cmd=cmd, passphrase=repoToGet['passphrase'])
            except Exception as e:
                self._to_console(f"ERROR: ({e}).")
                sys.exit(1)

        if 'borgbackup' in Config._CONFIG and len(Config._CONFIG['borgbackup'].keys()) > 0 and Config._CONFIG['borgbackup']["enabled"]:
            cmd = [
                "extract",
                Config._CONFIG['borgbackup']['repo'],
                f"{Config._CONFIG['global']['restore']['dir']}/borgbackup"
            ]
            try:
                self._cmd_run_borg(cmd=cmd, passphrase=item['passphrase'])
            except Exception as e:
                self._to_console(f"ERROR: ({e}).")
                sys.exit(1)

    def fuzz(self, count=10, verify=False):
        """
        fuzz around in your backups to check if random files are in the backup

        Parameters
        ----------
        count: int
            define the number of files to check

        verify: bool
            if using restic: verify the restore file. this is time consuming.

        """
        if 'restic' in Config._CONFIG and len(Config._CONFIG['restic'].keys()) > 0 and Config._CONFIG['restic']["enabled"]:
            self._to_console("fuzzing around in configured backups..")

            files_per_repo_target = {}
            try:
                while count > 0:
                    # get a random repo
                    randomrepo = random.choice(Config._CONFIG['restic']['repos'])
                    # get a random backup target from the random repo
                    randomdir = random.choice(randomrepo['targets'])
                    randomdir_hash = sha512(randomdir.encode('utf-8')).hexdigest()

                    if randomdir_hash not in files_per_repo_target:
                        self._to_console(f"preseeding files for {randomdir}. this might take a while..")
                        files_per_repo_target[randomdir_hash] = self._preseed_filelist(randomdir)

                    # get a random file from the random backup target from the random repo
                    randomfile = os.path.basename(random.choice(files_per_repo_target[randomdir_hash]))

                    if self.debug:
                        self._to_console(f"..in repo {randomrepo['repo']}")
                        self._to_console(f"..at dir {randomdir}")

                    self._to_console(f"...by checking for file {randomfile} [repo [{randomrepo['repo']}] / target [{randomdir}]]")

                    # restore the random file from the and so on
                    try:
                        self._restic_get_one_file(filename=randomfile, out_of=randomrepo, verify=verify)
                    except Exception as e:
                        self._to_console(f"ERROR: ({e}).")
                        sys.exit(1)
                    count = count - 1
            except Exception as e:
                self._to_console(f"ERROR: ({e}).")
                sys.exit(1)

    def _cmd_run_restic(self, cmd, passphrase):
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
        self._shell_env['RESTIC_PASSWORD'] = passphrase

        command = [Config._CONFIG['restic']['bin']]
        command.extend(cmd)

        if self.debug:
            self._to_console(f"{shlex.join(command)}")

        self._logfile_handle.write(f"{time.ctime(self._timestamp)} command: {shlex.join(command)} \n")

        try:
            result = subprocess.run(command, capture_output=True, env=self._shell_env, encoding='utf-8')
            self._to_console(result.stdout)

            if self.debug:
                self._to_console(result.stderr)

            if result.returncode != 0:
                raise Exception(f"{command} not succesful. {result.stderr}")
        except Exception as e:
            raise Exception(e)

        # handle log output
        self._logfile_handle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
        self._logfile_handle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

        return (result.returncode, result.stdout, result.stderr)

    def _cmd_run_borg(self, cmd, passphrase):
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
        self._shell_env['BORG_PASSPHRASE'] = passphrase

        command = [Config._CONFIG['borgbackup']['bin']]
        command.extend(cmd)

        if self.debug:
            self._to_console(f"{shlex.join(command)}")

        self._logfile_handle.write(f"{time.ctime(self._timestamp)} command: {shlex.join(command)} \n")

        try:
            result = subprocess.run(command, capture_output=True, env=self._shell_env, encoding='utf-8')
            self._to_console(result.stdout)

            if self.debug:
                self._to_console(result.stderr)

            if result.returncode != 0:
                raise Exception(f"{command} not succesful. {result.stderr}")
        except Exception as e:
            raise Exception(e)

        # handle log output
        self._logfile_handle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
        self._logfile_handle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

        return (result.returncode, result.stdout, result.stderr)

    def _do_rsync(self, source, target):
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
            self._to_console(f"{shlex.join(command)}")

        self._logfile_handle.write(f"{time.ctime(self._timestamp)} rsync command: {shlex.join(command)} \n")

        try:
            result = subprocess.run(command, capture_output=True, env=self._shell_env, encoding='utf-8')
            self._to_console(result.stdout)

            if self.debug:
                self._to_console(result.stderr)

            if result.returncode != 0:
                raise Exception(f"{command} not succesful. {result.stderr}")
        except Exception as e:
            raise Exception(e)

        # handle log output
        self._logfile_handle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
        self._logfile_handle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

    def _do_b2_sync(self, source, target):
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
            Config._CONFIG['b2']['binary'],
            'sync',
            '--keepDays',
            '7',
            source,
            target
        ]

        for item in Config._CONFIG['b2']['env']:
            self._shell_env[item['name']] = item['value']

        if self.debug:
            self._to_console(f"{shlex.join(command)}")

        self._logfile_handle.write(f"{time.ctime(self._timestamp)} b2 command: {shlex.join(command)} \n")

        try:
            result = subprocess.run(command, capture_output=True, env=self._shell_env, encoding='utf-8')
            self._to_console(result.stdout)

            if self.debug:
                self._to_console(result.stderr)

            if result.returncode != 0:
                raise Exception(f"{command} not succesful. {result.stderr}")
        except Exception as e:
            raise Exception(e)

        # handle log output
        self._logfile_handle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
        self._logfile_handle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

        return (result.returncode, result.stdout, result.stderr)

    def _to_console(self, message):
        """
        common function to print to console.
        """
        print(f"{message}")
        return True

    def _load_config(self, fle):
        """
        load the config from file
        """
        ymlcnt = None
        try:
            with open(fle) as file:
                ymlcnt = yaml.safe_load(file)
                file.close()
        except Exception as e:
            self._to_console(f"Error while parsing connfig ({fle}): {e}")
        return ymlcnt

    def _ask_repo_to_restore(self, lst):
        """
        get a list of repos and print a user facing selector.
        """
        selectedNumber = None
        self._to_console("Please select a repository: ")

        while True:
            for i, v in enumerate(lst):
                self._to_console(f" [{i}] - {v['repo']}")
            answer = input(f"--> select a repo [0 to {len(lst)-1}] from the list above or [c]ancel ".lower())

            if answer == "c":
                sys.exit(1)
            if int(answer) >= 0 and int(answer) <= len(lst) - 1:
                selectedNumber = answer
                break

        return selectedNumber

    def _restic_get_one_file(self, filename=None, out_of=None, verify=None):
        if filename is not None and out_of is not None:
            cmd = [
                "restore",
                "latest",
                "--repo",
                out_of['repo'],
                "--target",
                f"{Config._CONFIG['global']['restore']['dir']}/restic",
                '--include',
                filename
            ]

            if verify:
                cmd.append('--verify')
            try:
                self._cmd_run_restic(cmd=cmd, passphrase=out_of['passphrase'])
            except Exception as e:
                self._to_console(f"ERROR: ({e}).")
                sys.exit(1)

    def _preseed_filelist(self, randomdir=None):
        if randomdir is not None and os.path.isdir(randomdir):
            return [y for x in os.walk(randomdir) for y in glob(os.path.join(x[0], '*'))]
        elif os.path.isfile(random):
            return [randomdir]

        return randomdir
