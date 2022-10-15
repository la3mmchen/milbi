#!/usr/bin/env python3
import os
import sys
import shutil
import subprocess
"""
some basic tests that uses the cli
"""


def test_cli_broken():
    """
    test if cli can detect broken commands
    """
    cmd = _prepare()
    cmd['args'].extend(['b0rked'])
    assert _run(cmd) > 0


def test_cli():
    """
    test if cli can run without subcommands
    """
    cmd = _prepare()
    assert _run(cmd) == 0


def test_cli_help():
    """
    test if cli can print help
    """
    cmd = _prepare()
    cmd['args'].extend(['--help'])
    assert _run(cmd) == 0


def test_cli_config():
    """
    test if cli can access the config
    """
    cmd = _prepare()
    cmd['args'].extend(['config'])
    assert _run(cmd) == 0


def test_cli_explain():
    """
    test if cli can show the config explanation
    """
    cmd = _prepare()
    cmd['args'].extend(['config', '--explain'])
    assert _run(cmd) == 0


def test_cli_backup_simulate():
    """
    test if cli can do a simulated backup
    """
    cmd = _prepare()
    cmd['args'].extend(['backup', '--simulate'])
    assert _run(cmd) == 0


def test_cli_backup():
    """
    test if cli can execute a backup
    """
    cmd = _prepare()
    cmd['args'].extend(['backup'])
    assert _run(cmd) == 0


def test_cli_info():
    """
    test if cli can print info
    """
    cmd = _prepare()
    cmd['args'].extend(['info'])
    assert _run(cmd) == 0


def test_cli_check():
    """
    test if cli can execute the consistency checks
    """
    cmd = _prepare()
    cmd['args'].extend(['check'])
    assert _run(cmd) == 0


def test_cli_state():
    """
    test if cli can execute state subcmd
    """
    cmd = _prepare()
    cmd['args'].extend(['state'])
    assert _run(cmd) == 0


def test_cli_prune():
    """
    test if cli can prune from the repository
    """
    cmd = _prepare()
    cmd['args'].extend(['prune'])
    assert _run(cmd) == 0


def test_cli_sync():
    """
    test if cli can execute syncs
    """
    _init()

    cmd = _prepare()
    cmd['args'].extend(['sync'])
    assert _run(cmd) == 0

    _cleanup()


def test_cli_restore():
    """
    test if cli can execute restores
    """
    _init()

    cmd = _prepare()
    cmd['args'].extend(['get', '--repo', 'resticrepo'])
    assert _run(cmd) == 0

    _cleanup()


def test_cli_fuzz():
    """
    test if cli can execute fuzz testing
    """
    _init()

    cmd = _prepare()
    cmd['args'].extend(['fuzz', '--count', '3'])
    assert _run(cmd) == 0

    _cleanup()


def test_cli_flows():
    """
    test if cli can execute all flows
    """
    _init()

    cmd = _prepare()
    cmd['args'].extend(['flows'])
    assert _run(cmd) == 0

    _cleanup()


def test_cli_flow():
    """
    test if cli can execute one specific flow
    """
    _init()

    cmd = _prepare()
    cmd['args'].extend(['flows', '--flow', 'quick'])
    assert _run(cmd) == 0

    cmd = _prepare()
    cmd['args'].extend(['flows', '--flow', 'none'])
    assert _run(cmd) == 0

    cmd = _prepare()
    cmd['args'].extend(['flows', '--flow', 'wrong'])
    assert _run(cmd) == 0

    _cleanup()


"""
helpers
"""

def _init():
    try:
        for item in ['./tests/files/resticrepo-copy', './tests/files/resticrepo-restore']:
            if not os.path.exists(item):
                os.mkdir(item)
    except:
        pass

def _prepare():
    cmd = {
        'args': ['./milbi.py', '--config', './tests/files/clitest-config.yaml'],
    }
    return cmd


def _cleanup():
    for item in ['./tests/files/resticrepo-copy', './tests/files/resticrepo-restore']:
        if os.path.exists(item) and os.path.isdir(item):
            for files in os.listdir(item):
                path = os.path.join(item, files)
                try:
                    shutil.rmtree(path)
                except OSError:
                    os.remove(path)

def _run(cmd):
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
