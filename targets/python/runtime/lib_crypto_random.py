"""lib.crypto.random: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_random(t, *args):
    crypto_call(t, 'random', args, 'bytes')
