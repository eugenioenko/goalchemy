"""Byte-exact output to standard error and standard output."""

import sys


def write_stderr(b):
    sys.stderr.buffer.write(b)
    sys.stderr.buffer.flush()


def write_stdout(b):
    sys.stdout.buffer.write(b)
    sys.stdout.buffer.flush()
