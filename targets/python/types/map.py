"""Go maps: insertion-ordered entries with snapshot iteration."""


class Entry:
    __slots__ = ("k", "v", "live")

    def __init__(self, k, v):
        self.k = k
        self.v = v
        self.live = True


class GoMap:
    __slots__ = ("index", "entries", "key_of")

    def __init__(self, key_of):
        self.index = {}
        self.entries = []
        self.key_of = key_of

    def compact(self):
        if len(self.entries) > 32 and len(self.entries) > 2 * len(self.index):
            self.entries = [e for e in self.entries if e.live]


class MapIter:
    __slots__ = ("entries", "i", "k", "v")

    def __init__(self, entries):
        self.entries = entries
        self.i = 0
        self.k = None
        self.v = None


def identity_key(k):
    return k
