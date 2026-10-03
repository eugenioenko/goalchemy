"""lib.crypto.rsaoaep_decrypt: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_rsaoaep_decrypt(t, *args):
    crypto_call(t, 'rsaoaep_decrypt', args, 'bytes')
