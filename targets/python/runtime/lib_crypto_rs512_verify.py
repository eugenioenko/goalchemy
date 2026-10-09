"""lib.crypto.rs512_verify: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_rs512_verify(t, *args):
    crypto_call(t, 'rs512_verify', args, 'bool')
