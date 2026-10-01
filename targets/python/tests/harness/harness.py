"""Serves runtime conformance requests for the Python target over JSON
Lines on standard input and output."""

import json
import os
import sys

sys.path.insert(0, os.path.dirname(__file__))

from codec import rt  # noqa: E402
from harness_gen import CASES  # noqa: E402

PROTOCOL = 1


def serve(req):
    base = {"v": PROTOCOL, "id": req.get("id")}
    if req.get("v") != PROTOCOL:
        return dict(base, status="harness_failure", error="unsupported protocol version")
    fn = CASES.get(req.get("case"))
    if fn is None:
        return dict(base, status="harness_failure", error="unknown case %s" % req.get("case"))
    after = {}

    class H:
        def let(self, n):
            return req["let"][n]

        def after(self, n, v):
            after[n] = v

    try:
        rt.reset_scheduler()
        results = fn(H())
        return dict(base, status="returned", results=results, after=after)
    except rt.Blocked:
        return dict(base, status="blocked")
    except rt.GoPanic as p:
        return dict(base, status="panic", panic=rt.format_panic_value(p.value).decode("utf-8", "replace"))
    except Exception as e:  # noqa: BLE001 - reported as a harness failure
        return dict(base, status="harness_failure", error="%s: %s" % (type(e).__name__, e))


def main():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
        except ValueError as e:
            resp = {"v": PROTOCOL, "status": "harness_failure", "error": str(e)}
        else:
            resp = serve(req)
        sys.stdout.write(json.dumps(resp) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
