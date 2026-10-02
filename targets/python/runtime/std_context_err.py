"""Owner-bound source contexts with anchored absolute deadlines and pruning."""
from .chan_close import chan_close
from .chan_make import make_chan
from .std_errors_new import std_errors_new
from ..types.panic import nil_deref

CONTEXT_CANCELED = std_errors_new(b"context canceled")
CONTEXT_DEADLINE_EXCEEDED = std_errors_new(b"context deadline exceeded")


class Context:
    __slots__ = ("done", "err", "children", "parent", "owner", "deadline", "timer", "hooks", "__weakref__")

    def __init__(self, done, owner=None):
        self.done, self.owner = done, owner
        self.err = self.parent = self.deadline = self.timer = None
        self.children = set()
        self.hooks = set()
        if owner is not None:
            owner.contexts.add(self)

    def detach(self):
        from .task_spawn import sched, HostFault
        if self.owner is not None:
            self.owner.check()
            if sched() is not self.owner:
                raise HostFault("foreign source context cleanup")
        if self.parent is not None:
            self.parent.children.discard(self)
            self.parent = None
        if self.timer is not None:
            self.owner.remove_timer(self.timer)
            self.timer = None
        for child in list(self.children):
            child.detach()
        self.children.clear()
        self.hooks.clear()
        if self.owner is not None:
            self.owner.contexts.discard(self)

    def _add_hook(self, hook):
        # Internal remover is valid during inactive-owner retirement, but never
        # on a native callback or a later owner. Public removal also checks live.
        from .task_spawn import sched, HostFault
        check_context(self)
        owner = sched()
        if self.err is not None:
            hook()
        elif self is not BACKGROUND:
            # Background never cancels; retaining hooks would create a global
            # root outside this owner and make their removal impossible later.
            self.hooks.add(hook)
        def remove():
            if sched() is not owner:
                raise HostFault("foreign source context hook")
            self.hooks.discard(hook)
        return owner, remove

    def add_hook(self, hook):
        from .task_spawn import sched, HostFault
        owner, internal = self._add_hook(hook)
        def remove():
            if sched() is not owner or not owner.active:
                raise HostFault("foreign source context hook")
            internal()
        return remove


BACKGROUND = Context(None)


def check_context(c):
    from .task_spawn import HostFault, sched
    sched().check()
    if c is None:
        raise nil_deref()
    if type(c) is not Context or (c.owner is not None and (c.owner is not sched() or not c.owner.active)):
        raise HostFault("foreign source context")
    if c.owner is not None:
        c.owner.check()
    if c.err is None and c.deadline is not None and c.deadline <= sched().now():
        cancel_context(c, CONTEXT_DEADLINE_EXCEEDED)
    return c


def cancel_context(c, err):
    from .task_spawn import HostFault, sched
    sched().check()
    if c is None:
        raise nil_deref()
    if type(c) is not Context or (c.owner is not None and (c.owner is not sched() or not c.owner.active)):
        raise HostFault("foreign source context")
    if c.owner is not None:
        c.owner.check()
    if c.err is not None or c is BACKGROUND:
        return
    c.err = err
    chan_close(c.done)
    errors = []
    for child in list(c.children):
        try:
            cancel_context(child, err)
        except BaseException as e:
            errors.append(e)
    for hook in list(c.hooks):
        try:
            hook()
        except BaseException as e:
            errors.append(e)
    c.detach()
    if errors:
        raise HostFault("context cancel hook: " + "; ".join(str(e) for e in errors))


def new_child(parent):
    from .task_spawn import sched
    check_context(parent)
    c = Context(make_chan(0, lambda: None), sched())
    c.deadline = parent.deadline
    if parent.err is not None:
        cancel_context(c, parent.err)
    elif parent is not BACKGROUND:
        c.parent = parent
        parent.children.add(c)
    return c


def arm_deadline(c):
    if c.err is not None or c.deadline is None:
        return
    if c.deadline <= c.owner.now():
        cancel_context(c, CONTEXT_DEADLINE_EXCEEDED)
    else:
        c.timer = c.owner.add_timer_at(c.deadline, None, lambda: cancel_context(c, CONTEXT_DEADLINE_EXCEEDED))


def std_context_context_err(c):
    return check_context(c).err


def host_boundary(parent, duration):
    """Generic request-only timeout, anchored before copying/submission."""
    from .task_spawn import deadline, sched
    check_context(parent)
    at = deadline(sched().now(), duration)
    c = new_child(parent)
    c.deadline = at if c.deadline is None else min(at, c.deadline)
    arm_deadline(c)
    return c
