from __future__ import (absolute_import, division, print_function)
__metaclass__ = type


def test(sentinel):
    return sentinel == 'test'


class FilterModule(object):
    ''' Fake test plugin for ansible-operator '''

    def filters(self):
        return {
            'test': test
        }
