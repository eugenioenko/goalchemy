"""lib.crypto.rs256_verify: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_rs256_verify(t, *args):
    crypto_call(t, 'rs256_verify', args, 'bool')
