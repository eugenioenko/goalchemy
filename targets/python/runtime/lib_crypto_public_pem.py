"""lib.crypto.public_pem: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_public_pem(t, *args):
    crypto_call(t, 'public_pem', args, 'text')
