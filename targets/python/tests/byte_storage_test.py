"""Structural native-storage checks, run against the emitted common fixture."""

import gc
import main
import rt


def native(s):
    assert type(s.a) is bytearray
    assert s.b and len(s.a) >= s.o + s.c
    assert memoryview(s.a).itemsize == 1
    return s


def panic(f):
    try:
        f()
    except rt.GoPanic as p:
        assert p.value.t is rt.RUNTIME_ERROR
    else:
        raise AssertionError("expected source bounds/make panic")


def fault(f):
    try:
        f()
    except rt.Fault:
        pass
    else:
        raise AssertionError("expected allocation fault")


assert rt.BYTE_NIL.a is None and rt.BYTE_NIL.b
assert not rt.NIL.b
for empty in (rt.BYTE_NIL, rt.make_slice(0, 0, lambda: 0, True)):
    rt.clear_slice(empty, lambda: 99)
    assert rt.copy_string(empty, b"abc") == 0
    assert rt.copy_string(empty, b"") == 0
    assert (empty.a is None) == (empty is rt.BYTE_NIL)
    assert rt.copy(empty, rt.BYTE_NIL) == 0
    assert rt.copy(rt.BYTE_NIL, empty) == 0
    assert rt.append(empty, bytearray()) is empty
    assert rt.append_slice(empty, rt.BYTE_NIL) is empty
    assert rt.append_slice(empty, rt.to_bytes(b"")) is empty
    assert rt.append_string(empty, b"") is empty
    assert rt.reslice(empty, 0, 0, 0).b
    result = native(rt.append(rt.reslice(empty, 0, 0), bytearray((255, 128))))
    assert result.a == bytearray((255, 128))
assert rt.make_slice(0, 0, lambda: 0, True).a is not None
assert rt.slice_to_array(rt.BYTE_NIL, 0) == bytearray()
assert type(rt.slice_to_array(rt.BYTE_NIL, 0)) is bytearray

s = native(rt.make_slice(2, 6, lambda: 99, True))
assert s.a == bytearray(6)  # the entire capacity is physically zero
rt.sset(s, 0, 255)
rt.sset(s, 1, 128)
assert rt.sget(s, 0) == 255 and type(rt.sget(s, 0)) is int
rt.clear_slice(rt.reslice(s, 1, 1), lambda: 99)
rt.clear_slice(rt.reslice(s, 6, 6), lambda: 99)
assert s.a == bytearray((255, 128, 0, 0, 0, 0))
view = rt.reslice(s, 1, 2, 5)
r = native(rt.append(view, bytearray((3, 4))))
assert (s.o, s.l, s.c) == (0, 2, 6)
assert (view.o, view.l, view.c) == (1, 1, 4)
assert (r.o, r.l, r.c) == (1, 3, 4) and r.a is s.a
assert len(s.a) == 6 and s.a == bytearray((255, 128, 3, 4, 0, 0))
grown = native(rt.append_slice(r, r))
assert grown.a is not s.a and (grown.o, grown.l, grown.c) == (0, 6, 8)
assert grown.a == bytearray((128, 3, 4, 128, 3, 4, 0, 0))
rt.sset(grown, 0, 9)
assert rt.sget(r, 0) == 128

s = native(rt.to_bytes(bytes((1, 2, 3, 4, 5, 6))))
a = s.a
assert rt.copy(rt.reslice(s, 1), s) == 5
assert a == bytearray((1, 1, 2, 3, 4, 5))
assert rt.copy(s, rt.reslice(s, 2)) == 4
assert a == bytearray((2, 3, 4, 5, 4, 5))
for dest, src in ((0, 1), (1, 0)):
    b = native(rt.to_bytes(bytes((1, 2, 3, 4, 5, 6))))
    result = native(rt.append_slice(rt.reslice(b, dest, dest + 2), rt.reslice(b, src, src + 3)))
    assert result.a is b.a and len(b.a) == 6
    # Compare against an independent immutable source snapshot.
    oracle = bytearray((1, 2, 3, 4, 5, 6))
    oracle[dest + 2:dest + 5] = bytes((1, 2, 3, 4, 5, 6))[src:src + 3]
    assert b.a == oracle

# Distinct buffers retain byte offsets and owned target storage. Mutating the
# destination afterwards cannot change the source of either copy or append.
source = native(rt.to_bytes(b"x\0\xff\x80y"))
destination = native(rt.make_slice(3, 8, lambda: 0, True))
assert rt.copy(destination, rt.reslice(source, 1, 4)) == 3
assert rt.from_bytes(destination) == b"\0\xff\x80"
combined = native(rt.append_slice(destination, rt.reslice(source, 1, 4)))
assert combined.a is destination.a and rt.from_bytes(combined) == b"\0\xff\x80\0\xff\x80"
rt.sset(combined, 0, 9)
assert rt.from_bytes(source) == b"x\0\xff\x80y"
saved = rt.from_bytes(rt.reslice(combined, 1, 4))
rt.sset(combined, 1, 8)
assert saved == b"\xff\x80\0"
rt.clear_slice(rt.reslice(s, 1, 4), lambda: 99)
assert a == bytearray((2, 0, 0, 0, 4, 5)) and len(a) == 6

binary = b"\x00\xff\x80\xc0\xafA"
s = native(rt.to_bytes(binary))
saved = rt.from_bytes(s)
rt.sset(s, 0, 44)
assert saved == binary and rt.from_bytes(s) != binary
other = native(rt.to_bytes(saved))
rt.sset(other, 1, 1)
assert saved == binary and s.a[1] == 255
b = native(rt.make_slice(0, 10, lambda: 0, True))
b = native(rt.append_string(b, binary))
assert b.a == bytearray(binary + bytes(4))
assert rt.copy_string(rt.reslice(b, 1), binary) == 5
assert len(b.a) == 10

# Invoke actual emitted array helpers, including zero-length/named arrays.
arrays = []
for name, zero in vars(main).items():
    if name.startswith("zero_") and callable(zero):
        a = zero()
        if type(a) is not bytearray:
            continue
        arrays.append(a)
        suffix = name[5:]
        clone = getattr(main, "clone_" + suffix)
        set_array = getattr(main, "set_" + suffix)
        eq = getattr(main, "eq_" + suffix)
        key = getattr(main, "key_" + suffix)
        assert memoryview(a).itemsize == 1 and a == bytearray(len(a))
        copied = clone(a)
        assert type(copied) is bytearray and copied is not a and eq(a, copied)
        snapshot = key(a)
        assert type(snapshot) is bytes
        sliced = native(rt.slice_array(a))
        if a:
            copied[0] = 255
            assert a[0] == 0 and not eq(a, copied)
            set_array(a, copied)
            assert a is sliced.a and a[0] == 255 and snapshot[0] == 0
            converted = rt.slice_to_array(sliced, len(a))
            assert type(converted) is bytearray and converted is not a
            converted[0] = 128
            assert a[0] == 255
        set_array(a, a)
assert arrays and any(len(a) == 4 for a in arrays) and any(len(a) == 0 for a in arrays)

s = native(rt.to_bytes(b"ab"))
original = bytes(s.a)
for index in (-1, 2, 1 << 63, (1 << 64) - 1):
    panic(lambda: rt.sget(s, index))
    panic(lambda: rt.sset(s, index, 9))
assert bytes(s.a) == original
panic(lambda: rt.reslice(s, 0, 1 << 63))
panic(lambda: rt.slice_to_array(s, 3))
for n, c in ((-1, 2), (3, 2), (1 << 63, 1 << 63), (1, 1 << 63)):
    panic(lambda: rt.make_slice(n, c, lambda: 0, True))
fault(lambda: rt.make_slice(0, 1 << 32, lambda: 0, True))
# Size validation must happen before reading or writing enormous synthetic ranges.
fake = rt.Slice(s.a, 0, 2**31, 2**31, True)
fault(lambda: rt.append(fake, bytearray((1,))))
assert bytes(s.a) == original

ints = rt.make_slice(2, 4, lambda: -1)
assert type(ints.a) is list and not ints.b and ints.a == [-1] * 4
ints = rt.append(ints, [1000])
assert ints.a == [-1, -1, 1000, -1]
rt.copy(rt.reslice(ints, 1), ints)
assert ints.a[:3] == [-1, -1, -1]
assert rt.from_runes(rt.to_runes("é".encode())) == "é".encode()
# Native backing follows ordinary reachability when only a view survives.
def escaped():
    return rt.reslice(rt.to_bytes(binary), 1)
escaped_view = escaped()
gc.collect()
assert rt.from_bytes(escaped_view) == binary[1:]
print("Python native byte storage checks passed")
