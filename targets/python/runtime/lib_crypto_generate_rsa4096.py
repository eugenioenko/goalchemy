"""lib.crypto.generate_rsa4096: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_generate_rsa4096(t, *args):
    crypto_call(t, 'generate_rsa4096', args, 'key')
