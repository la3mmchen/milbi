#!/usr/bin/env python3

import os
import yaml
import typing


class Config(object):
    _CONFIG_FILE: typing.Optional[str] = None
    _CONFIG: typing.Optional[dict] = None

    # define required attributes
    _GLOBAL: typing.Optional[str] = None
    _BORGBACKUP: typing.Optional[str] = None
    _RESTIGBACKUP: typing.Optional[str] = None
    _B2: typing.Optional[str] = None

    _ATTRIBUTES = {
        'global': {
            'logfile': '<String>; Provide a path in your filesystem for the application to log.',
            'debug': 'Set to true to enable debug output.',
            'statefile': '<String> Provide a path in your filesystem for the application to keep a rudimental state.',
            'hostalias': '<String> Provide a hostname',
            'restore': {
                'dir': '<String> Provide a path in your filesystem for the application to restore',
            }
        },
        'borgbackup': {
            'enabled': 'A trigger to activate/deactivate the backup for borgbackup. (Options: True / False).',
            'bin': 'Path to borg binary to use.',
            'repos': [
                {
                    'repo': 'Path to the existing borg repository.',
                    'passphrase': 'Passphrase for the repository',
                    'keep': "For prune actions: How many days to keep in the repository; e.g. 2 for 2 days",
                    'excludes': "A pattern for files to be excluded from the repository; e.g. *github*",
                    'targets': [
                        '/local/path',
                        '/local/path1'
                    ]
                }
            ]
        },
        'restic': {
            'enabled': 'A trigger to activate the backup with restic. (Options: True / False).',
            'bin': 'Path to borg binary to use.',
            'repos': [
                {
                    'repo': '~/backup # Path to the existing restic repository.',
                    'passphrase': 'Passphrase for the repository',
                    'keep': "For prune actions: How many days to keep in the repository; e.g. 3 for 3 days",
                    'excludes': "A pattern for files to be excluded from the repository; e.g. *github*",
                    'targets': [
                        '/local/path',
                        '/local/path1'
                    ]
                }
            ]
        },
        'syncs': [
            {
                'name': 'Give a meaningful name',
                'type': 'Define the technologie to use (options ...)',
                'source': 'A local path to a directory that you want to sync.',
                'target': 'The destination of the sync; might be a local path or maybe a b2 bucket.'
            }
        ],
        'b2': {
            'binary': 'Path to b2 binary to use.',
            'env': [
                {
                    'name': 'B2_APPLICATION_KEY_ID',
                    'value': '<application key_id from backblaze>'
                },
                {
                    'name': 'B2_APPLICATION_KEY',
                    'value': '<b2 app key>'
                }
            ]
        },
        'flows': [{
            'name': 'Give a meaningful name',
            'tasks': [
                'a milbi subcommand like backup,check,prune,sync',
                'a milbi subcommand like backup,check,prune,sync',
            ]
        }]
    }

    def __init__(self, config_file=None):
        if config_file is None:
            config_file = Config.get_required_env_var("CONFIG_FILE")

        config_file = Config.make_path_absolute(config_file)

        assert os.path.exists(config_file)

        Config._CONFIG_FILE = config_file
        with open(config_file, 'r') as f:
            Config._CONFIG = yaml.safe_load(f)

        # handle relative paths
        Config._CONFIG['global']['logfile'] = Config.make_path_absolute(Config._CONFIG['global']['logfile'])
        Config._CONFIG['global']['restore']['dir'] = Config.make_path_absolute(Config._CONFIG['global']['restore']['dir'])

    @staticmethod
    def get_config_file() -> str:
        return Config._CONFIG_FILE

    @staticmethod
    def get_required_env_var(envvar: str) -> str:
        if envvar not in os.environ:
            raise Exception("Please set the {envvar} environment variable")
        return os.environ[envvar]

    @staticmethod
    def get_required_config_var(configvar: str) -> str:
        assert Config._CONFIG
        if configvar not in Config._CONFIG:
            raise Exception(f"Please set the {configvar} variable in the config file {Config._CONFIG_FILE}")
        return Config._CONFIG[configvar]

    @staticmethod
    def make_path_absolute(inPath):
        """
        function to make sure a path is an absolute path
        """

        if '~' in inPath:
            inPath = os.path.expanduser(inPath)
        if not os.path.isabs(inPath):
            inPath = os.path.join(os.getcwd(), inPath)
        absPath = inPath

        return absPath

    @classmethod
    def get_some_var(cls) -> str:
        """Example variable that is set in the config file (preferred)"""
        if cls._B2 is None:
            cls._B2 = Config.get_required_config_var('b2')
        return cls._B2
