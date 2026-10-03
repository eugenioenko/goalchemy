"""lib.crypto.generate_p256: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_generate_p256(t, *args):
    crypto_call(t, 'generate_p256', args, 'key')
