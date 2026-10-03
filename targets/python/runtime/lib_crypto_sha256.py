"""lib.crypto.sha256: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_sha256(t, *args):
    crypto_call(t, 'sha256', args, 'bytes')
