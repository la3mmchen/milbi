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
            self._toConsole(f"---")
            self._toConsole(f"loaded config: ")
            for item in vars(self).items():
                self._toConsole("`-  %s: %s" % item)
            self._toConsole(f"---")

        if self._getContent() is None:
            self._toConsole(f"ERROR: no config file found.")
            sys.exit(1)

        self._shellEnv = os.environ.copy()


    def __del__(self):
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


    def sync(self):
        """
        execute configured syncs

        Parameters
        ----------

        """
        if 'syncs' in self._content and len(self._content['syncs']) > 0:
            for syncconfig in self._content['syncs']:
                if os.path.exists(syncconfig['target']):
                    try:
                        self._toConsole(f"syncing {syncconfig['name']} {syncconfig['source']} -> {syncconfig['target']}")
                        self._doRsync(syncconfig['source'], syncconfig['target'])
                    except Exception as e:
                        self._toConsole(f"ERROR: ({e}).")
                        sys.exit(1)
                else:
                    self._toConsole(f"{syncconfig['name']}: target directory not found. skipping")
                    pass


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
        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            cmd = [
                "backup",
                "--verbose",
                "--compression",
                "auto",
                "--repo",
                self._content['restic']['repo'],
                "--exclude",
                self._content['restic']['excludes'],
            ]
            cmd.extend(self._content['borgbackup']['targets'])

            try:
                self._cmdRunRestic(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)

        if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
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

        for restic:
          - check the repo

        Parameters
        ----------

        """
        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            cmd = [
                "check",
                "--repo",
                self._content['restic']['repo'],
            ]

            try:
                self._cmdRunRestic(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)

        if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
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
        show info of the existing repos (snapshots, repos, ...)

        Parameters
        ----------

        """
        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            cmd = [
                "snapshots",
                "--repo",
                self._content['restic']['repo'],
            ]

            try:
                self._cmdRunRestic(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)

        if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
            cmd = [
                "info",
                self._content['borgbackup']['repo']
            ]
            try:
                self._cmdRunBorg(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)

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
        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:
            cmd = [
                "forget",
                f"--keep-daily={self._content['restic']['keep']}",
                "--repo",
                self._content['restic']['repo']
            ]

            print(f"{cmd}")
            print(f"{shlex.join(cmd)}")
            try:
                self._cmdRunRestic(cmd)
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
                self._cmdRunBorg(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)


    def get(self, filter=None):
        """
        open backup (either by mounting or restoring)

        Parameters
        ----------
        filter: string
            only extract certain paths of backup (e.g. *ssh*)
        """
        if self._content['global']['restore']['dir']:
            if not os.path.exists(os.path.dirname(self._content['global']['restore']['dir'])):
                self._toConsole(f"ERROR: target directory {self._content['global']['restore']['dir']} does not exists.")

        if 'restic' in self._content and len(self._content['restic'].keys()) > 0 and self._content['restic']["enabled"]:

            cmd = [
                "restore",
                "latest",
                "--repo",
                self._content['restic']['repo'],
                "--target",
                f"{self._content['global']['restore']['dir']}/restic"
            ]

            if filter is not None:
                cmd.extend([
                  '--path',
                  filter
                ])

            try:
                self._cmdRunRestic(cmd)
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
                self._cmdRunBorg(cmd)
            except Exception as e:
                self._toConsole(f"ERROR: ({e}).")
                sys.exit(1)


    def close(self):
      """
      close open backup

      Parameters
      ----------
      """

      if self._content['global']['restore']['dir']:
          if not os.path.exists(os.path.dirname(self._content['global']['restore']['dir'])):
              self._toConsole(f"ERROR: target directory {self._content['global']['restore']['dir']} does not exists.")

      if 'borgbackup' in self._content and len(self._content['borgbackup'].keys()) > 0 and self._content['borgbackup']["enabled"]:
          cmd = [
              "umount",
              self._content['global']['restore']['dir']
          ]
          try:
              self._cmdRunBorg(cmd)
          except Exception as e:
              self._toConsole(f"ERROR: ({e}).")
              sys.exit(1)


    def _cmdRunRestic(self, cmd):
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
        if "passphrase" in  self._content['restic']:
            self._shellEnv['RESTIC_PASSWORD'] = self._content['restic']['passphrase']

        command = [ self._content['restic']['bin'] ]
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
        if "passphrase" in  self._content['borgbackup']:
            self._shellEnv['BORG_PASSPHRASE'] = self._content['borgbackup']['passphrase']

        command = [ self._content['borgbackup']['bin'] ]
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
