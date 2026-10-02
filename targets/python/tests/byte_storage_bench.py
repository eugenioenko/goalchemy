"""Fresh-process native/generic byte payload memory and throughput diagnostic.

Run from the Goalchemy root: python3 targets/python/tests/byte_storage_bench.py native 2
Repeat native/boxed with 2/8 MiB. No thresholds are used by mandatory tests.
"""

import gc
import json
import resource
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from python import rt_index as rt


def measure(kind, mib):
    native = kind == "native"
    n = mib * 1024 * 1024
    # Warm the selected path without retaining any warm-up payload.
    warm = rt.make_slice(32, 32, lambda: 0, native)
    rt.copy(warm, warm)
    rt.append_slice(warm, warm)
    del warm
    gc.collect()
    src = rt.make_slice(n, n, lambda: 0, native)
    dst = rt.make_slice(n, n, lambda: 0, native)
    for i in range(n):
        src.a[i] = i & 255
    start = time.perf_counter()
    for _ in range(16):
        assert rt.copy(dst, src) == n
    copy_ms = (time.perf_counter() - start) * 1000
    start = time.perf_counter()
    grown = rt.append_slice(src, src)
    append_ms = (time.perf_counter() - start) * 1000
    gc.collect()
    buffers = (src.a, dst.a, grown.a)
    assert all(type(a) is (bytearray if native else list) for a in buffers)
    assert grown.a is not src.a and grown.l == grown.c == 2 * n
    for a in buffers:
        assert a[0] == 0 and a[n - 1] == 255
    assert grown.a[n] == 0 and grown.a[2 * n - 1] == 255
    print(json.dumps({
        "storage": kind, "input_mib": mib,
        "native_backing_bytes": sum(len(a) for a in buffers) if native else None,
        "retained_backing_getsizeof_bytes": sum(sys.getsizeof(a) for a in buffers),
        "peak_rss_kib": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
        "copy_16_ms": round(copy_ms, 3), "append_ms": round(append_ms, 3),
        "python": sys.version.split()[0],
    }))


if __name__ == "__main__":
    if len(sys.argv) != 3 or sys.argv[1] not in ("native", "boxed") or sys.argv[2] not in ("2", "8"):
        raise SystemExit("usage: byte_storage_bench.py native|boxed 2|8")
    measure(sys.argv[1], int(sys.argv[2]))
