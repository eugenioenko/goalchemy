"""lib.crypto.rs384_sign: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_rs384_sign(t, *args):
    crypto_call(t, 'rs384_sign', args, 'bytes')
