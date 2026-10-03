"""lib.crypto.rs256_sign: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_rs256_sign(t, *args):
    crypto_call(t, 'rs256_sign', args, 'bytes')
