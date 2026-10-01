"""Imports the whole Python runtime as one namespace, for the harness."""
# flake8: noqa
import importlib
import os

_here = os.path.dirname(__file__)
for _dir in ("types", "runtime"):
    for _f in sorted(os.listdir(os.path.join(_here, _dir))):
        if _f.endswith(".py") and _f != "__init__.py":
            _m = importlib.import_module("python.%s.%s" % (_dir, _f[:-3]))
            globals().update({k: v for k, v in vars(_m).items() if not k.startswith("_")})
