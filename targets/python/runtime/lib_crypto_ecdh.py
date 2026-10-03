"""lib.crypto.ecdh: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_ecdh(t, *args):
    crypto_call(t, 'ecdh', args, 'bytes')
