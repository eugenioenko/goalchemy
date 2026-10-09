"""lib.crypto.es512_sign: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_es512_sign(t, *args):
    crypto_call(t, 'es512_sign', args, 'bytes')
