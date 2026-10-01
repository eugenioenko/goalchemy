"""std.sync.waitgroup.done: Add(-1)."""

from .std_sync_waitgroup_add import std_sync_waitgroup_add


def std_sync_waitgroup_done(wg):
    std_sync_waitgroup_add(wg, -1)
