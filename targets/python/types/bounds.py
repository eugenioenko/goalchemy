"""Go slice-bounds checks and messages."""

from .panic import runtime_panic


def _oob(msg):
    raise runtime_panic("slice bounds out of range " + msg)


def check2(lo, hi, limit, word):
    if hi < 0:
        _oob("[:%d]" % hi)
    if hi > limit:
        _oob("[:%d] with %s %d" % (hi, word, limit))
    if lo < 0:
        _oob("[%d:]" % lo)
    if lo > hi:
        _oob("[%d:%d]" % (lo, hi))


def check3(lo, hi, mx, limit, word):
    if mx < 0:
        _oob("[::%d]" % mx)
    if mx > limit:
        _oob("[::%d] with %s %d" % (mx, word, limit))
    if hi < 0:
        _oob("[:%d:]" % hi)
    if hi > mx:
        _oob("[:%d:%d]" % (hi, mx))
    if lo < 0:
        _oob("[%d::]" % lo)
    if lo > hi:
        _oob("[%d:%d:]" % (lo, hi))
