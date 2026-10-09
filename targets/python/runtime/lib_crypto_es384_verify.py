"""lib.crypto.es384_verify: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_es384_verify(t, *args):
    crypto_call(t, 'es384_verify', args, 'bool')
