"""lib.crypto.rs512_sign: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_rs512_sign(t, *args):
    crypto_call(t, 'rs512_sign', args, 'bytes')
