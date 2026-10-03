"""lib.crypto.rsaoaep_encrypt: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_rsaoaep_encrypt(t, *args):
    crypto_call(t, 'rsaoaep_encrypt', args, 'bytes')
