"""Canonical value encoding for the Python harness."""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", ".."))

from python import rt_index as rt  # noqa: E402

WIDE = {"i64", "u64"}


def dec_int(raw, kind):
    return int(raw)


def dec_bool(raw):
    return raw == "true"


def dec_string(raw):
    if "str" in raw:
        return raw["str"].encode("utf-8")
    return bytes.fromhex(raw["hex"])


def dec_error(raw):
    if raw.get("nil"):
        return None
    return rt.std_errors_new(raw["error"].encode("utf-8"))


def dec_slice(raw, dec, zero):
    if raw.get("nil"):
        return rt.NIL
    items = [dec(v) for v in raw["slice"]]
    c = int(raw["cap"]) if "cap" in raw else len(items)
    a = items + [zero() for _ in range(c - len(items))]
    return rt.Slice(a, 0, len(items), c)


def view(base, raw):
    lo = int(raw["lo"]) if "lo" in raw else 0
    hi = int(raw["hi"]) if "hi" in raw else base.l
    mx = int(raw["max"]) if "max" in raw else base.c
    return rt.Slice(base.a, base.o + lo, hi - lo, mx - lo)


def dec_map(raw, dk, dv):
    if raw.get("nil"):
        return None
    m = rt.GoMap(rt.identity_key)
    for e in raw["map"]:
        rt.map_set(m, dk(e["key"]), dv(e["value"]))
    return m


def dec_chan(raw, dec, zero):
    if raw.get("nil"):
        return None
    ch = rt.make_chan(int(raw.get("cap", "0")), zero)
    for v in raw["chan"]:
        ch.buf.append(dec(v))
    if raw.get("closed"):
        ch.closed = True
    return ch


def enc_int(v):
    return str(v)


def enc_bool(v):
    return "true" if v else "false"


def enc_string(v):
    return {"hex": bytes(v).hex()}


def enc_error(v):
    if v is None:
        return {"nil": True}
    return {"error": v.t.methods["Error"](v.v).decode("utf-8", "replace")}


def enc_slice(s, enc):
    if s.a is None:
        return {"nil": True}
    return {"slice": [enc(s.a[s.o + i]) for i in range(s.l)], "cap": str(s.c)}


def enc_array(a, enc):
    return {"array": [enc(v) for v in a]}


def enc_map(m, ek, ev):
    if m is None:
        return {"nil": True}
    it = rt.map_iter(m)
    items = []
    while rt.map_next(it):
        items.append({"key": ek(it.k), "value": ev(it.v)})
    return {"map": items}


def enc_chan(ch, enc):
    if ch is None:
        return {"nil": True}
    return {"chan": [enc(v) for v in ch.buf], "cap": str(ch.size), "closed": ch.closed}


def enc_zero(_):
    return {"zero": True}


class _SpawnCheck(rt.Frame):
    """Reports whether a spawned task ran before the parent yielded."""

    def __init__(self):
        super().__init__()
        self.ran = False
        self.before = False

    def step(self, t):
        if self.pc == 0:
            def mark():
                self.ran = True
                return []

            rt.spawn(rt.sync(mark))
            self.before = self.ran
            self.pc = 1
            rt.std_runtime_gosched(t)
            return
        if not self.ran:
            raise RuntimeError("spawned task did not run after the parent yielded")
        rt.ret(t, self)

    def results(self):
        return [self.before]


def new_spawn_check():
    return _SpawnCheck()


def harness_select2(t, a, b, dflt):
    rt.select(t, [(a, False, None), (b, False, None)], dflt)
