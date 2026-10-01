"""core.map.make: a new empty map."""

from ..types.map import GoMap


def make_map(key_of):
    return GoMap(key_of)
