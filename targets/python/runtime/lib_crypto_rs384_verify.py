"""lib.crypto.rs384_verify: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_rs384_verify(t, *args):
    crypto_call(t, 'rs384_verify', args, 'bool')
