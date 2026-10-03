"""lib.crypto.private_pem: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_private_pem(t, *args):
    crypto_call(t, 'private_pem', args, 'text')
