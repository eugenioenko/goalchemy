"""lib.crypto.generate_p521: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_generate_p521(t, *args):
    crypto_call(t, 'generate_p521', args, 'key')
