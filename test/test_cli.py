#!/usr/bin/env python3
import sys
import yaml
import subprocess
"""
some basic tests that uses the cli
"""


def test_cli_help():
    """
    test if cli can print help
    """
    cmd = prepare()
    cmd['args'].extend(['--help'])
    cmd['rc'] = 0
    assert run(cmd) == cmd['rc']


def test_cli_config():
    cmd = prepare()
    cmd['args'].extend(['config'])
    cmd['rc'] = 0
    assert run(cmd) == cmd['rc']


"""
helpers
"""


def getTestConfig():
    s = yaml.safe_load(
        """
        global:
          debug: True
          logfile: "~/.milbi/milbibackup.log"
          hostalias: "binky"
          restore:
              dir: "~/mnt"
        """
    )
    return s


def prepare():
    cmd = {
        'args': ['./milbi.py'],
    }
    return cmd


def run(cmd):
    try:
        process = subprocess.Popen(
            cmd['args'], shell=False, stdin=None)
    except Exception as e:
        print(f"ERROR running {cmd}")
        print(f"ERROR {format(sys.exc_info()[1])} \n{e}")
        return 1
    while True:
        if process.poll() is not None:
            break
    rc = process.poll()
    return rc
