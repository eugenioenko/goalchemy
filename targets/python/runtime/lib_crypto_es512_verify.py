"""lib.crypto.es512_verify: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_es512_verify(t, *args):
    crypto_call(t, 'es512_verify', args, 'bool')
