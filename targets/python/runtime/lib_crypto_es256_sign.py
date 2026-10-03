"""lib.crypto.es256_sign: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_es256_sign(t, *args):
    crypto_call(t, 'es256_sign', args, 'bytes')
