#!/usr/bin/env python3

import os
import json
import typing


class State(object):
    _state: typing.Optional[dict] = {
        'current': {},
        'previous': {},
    }
    _state_file = None
    _read_only = True

    def __init__(self, state_file=None, read_only=True):

        self._read_only = read_only
        self._state_file = state_file

        if not os.path.exists(state_file):
            open(state_file, 'a').close()

        with open(state_file) as f:
            try:
                old_one = json.load(f)
                self._state['previous'] = old_one['current']
            except Exception:
                self._state['previous'] = '{ }'

    def __del__(self):
        if not self._read_only:
            file_handle = open(self._state_file, "w")
            file_handle.write(f"{json.dumps(self._state)}")
            file_handle.close()

    def set_state_readwrite(self) -> bool:
        self._read_only = False
        return self._read_only

    def get_state(self) -> str:
        return self._state

    def set_state(self, to_add) -> bool:
        self._state['current'].update(to_add)

        return True
