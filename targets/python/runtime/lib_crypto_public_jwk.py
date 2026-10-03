"""lib.crypto.public_jwk: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_public_jwk(t, *args):
    crypto_call(t, 'public_jwk', args, 'strings')
