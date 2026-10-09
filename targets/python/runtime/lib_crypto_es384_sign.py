"""lib.crypto.es384_sign: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_es384_sign(t, *args):
    crypto_call(t, 'es384_sign', args, 'bytes')
