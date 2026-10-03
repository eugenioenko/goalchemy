"""lib.crypto.generate_rsa2048: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_generate_rsa2048(t, *args):
    crypto_call(t, 'generate_rsa2048', args, 'key')
