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
    handles local backup & to some point supports the restore process.

    for me it is at least better then the random bash scripts i had before.

    Parameters
    ----------
    config: string
        file of local config

    debug: bool
        print additional output

    logfile: string
        if set log all actions to this file

    """
    config = None
    debug = None
    logfile = None

    _content = None
    _fileLocation = None
    _logfileHandle = None
    _timestamp = None
    _shellEnv = None


    def __init__(self, config="config.yaml", debug=False, logfile=None):
        self.debug = debug
        self.logfile = logfile

        self._timestamp = calendar.timegm(time.gmtime())

        # handle relative paths

        # .. for config
        if not os.path.isabs(config):
            self._fileLocation = os.path.join(os.getcwd(), config)
        else:
            self._fileLocation = config

        # .. for logfile
        if logfile is not None and '~' in logfile:
            logfile = os.path.expanduser(logfile)
        if logfile is not None and not os.path.isabs(logfile):
            self.logfile = os.path.join(os.getcwd(), logfile)
        else:
            self.logfile = logfile

        if self.logfile is not None:
              print(f"logging to {self.logfile}")
              self._logfileHandle = open(self.logfile, "a")

        self._content = self._loadConfig(self._fileLocation)

        if self.debug:
            self._toConsole(f"---")
            self._toConsole(f"loaded config: ")
            for item in vars(self).items():
                self._toConsole("`-  %s: %s" % item)
            self._toConsole(f"---")

        if self._getContent() is None:
            self._toConsole(f"ERROR: no config file found.")
            sys.exit(1)

        self._shellEnv = os.environ.copy()
        if "passphrase" in  self._content['borgbackup']:
            self._shellEnv['BORG_PASSPHRASE'] = self._content['borgbackup']['passphrase']


    def __del__(self):
        if self.logfile is not None:
            self._logfileHandle.close()

    def _toConsole(self, message):
        print(f"{message}")
        return True


    def _loadConfig(self, fle):
        ymlcnt = None
        try:
            with open(fle) as file:
                ymlcnt = yaml.safe_load(file)
                file.close()
        except Exception as e:
            self._toConsole(f"Error while parsing connfig (%s): %s" % (fle, e))
        return ymlcnt

    def _getContent(self):
        return self._content

    def _borgCmd(self, subcmd):
        cmd = [
            self._content['borgbackup']['bin'],
            subcmd
        ]

        return cmd

    def config(self):
        """
        Shows the current config.

        Parameters
        ----------

        """
        if self._content is None:
            self._toConsole("ERROR no valid config found")
        else:
            self._toConsole(yaml.dump(self._getContent()))


    def backup(self, simulate=False):
        """
        Executes a backup

        Parameters
        ----------
        simulate: bool
            if true simulates a backup if this is possible
        """
        if 'borgbackup' in self._content:
            cmd = [
                "create",
                "--compression",
                "zlib,6",
                "--exclude",
                self._content['borgbackup']['excludes'],
            ]

            if simulate:
                cmd.append("--dry-run")

            cmd.append(f"{self._content['borgbackup']['repo']}::{self._timestamp}")
            cmd.extend(self._content['borgbackup']['targets'])

            try:
                self._cmdRunBorg(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)


    def check(self):
        """
        check backups if possible

        if borgbackup is configured:
          - execute borg check on the last archive
          - run dry-run extract on the last archive

        Parameters
        ----------

        """
        if 'borgbackup' in self._content:
            #
            # execute borg check on repository
            #
            self._logfileHandle.write(f"{time.ctime(self._timestamp)} running borg check \n")
            cmd = [
                "check",
                "--verify-data",
                "--last",
                "1",
                self._content['borgbackup']['repo']
            ]

            try:
                self._cmdRunBorg(cmd)
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
                _, stdout, _ = self._cmdRunBorg(cmd)
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
                self._cmdRunBorg(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)


    def info(self):
        """
        show info of the existing backups

        Parameters
        ----------

        """
        if 'borgbackup' in self._content:
            cmd = [
                "info",
                self._content['borgbackup']['repo']
            ]
            try:
                self._cmdRunBorg(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)


    def list(self):
        """
        show info for configured backups

        Parameters
        ----------

        """
        if 'borgbackup' in self._content:
            cmd = [
                "list",
                self._content['borgbackup']['repo']
            ]
            try:
                self._cmdRunBorg(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)


    def prune(self):
        """
        if possible prune backups to

        Parameters
        ----------

        """
        if 'borgbackup' in self._content:
            cmd = [
                "prune",
                "--verbose",
                "--list",
                f"--keep-daily={self._content['borgbackup']['keep']}",
                self._content['borgbackup']['repo']
            ]
            try:
                self._cmdRunBorg(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)

    def _getLastArchive(self):
        """
        returns identifier for the last
        """

    def _cmdRunBorg(self, cmd):
        """
        generic function to run borg commands.

        Parameters
        ----------
        cmd: string
            Shell command to execute

        Returns
        -------
        result.returncode: int
            return code of command
        stdout: list
            outputs of command
        stderr: list
            error output of command
        """
        command = [ self._content['borgbackup']['bin'] ]
        command.extend(cmd)

        if self.debug:
            print(f"{shlex.join(command)}")

        if self.logfile is not None:
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
        if self.logfile is not None:
            self._logfileHandle.write(f"{time.ctime(self._timestamp)} stdout: {result.stdout} \n")
            self._logfileHandle.write(f"{time.ctime(self._timestamp)} stderr: {result.stderr} \n")

        return (result.returncode, result.stdout, result.stderr)