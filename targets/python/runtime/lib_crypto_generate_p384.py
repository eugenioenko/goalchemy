"""lib.crypto.generate_p384: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_generate_p384(t, *args):
    crypto_call(t, 'generate_p384', args, 'key')
