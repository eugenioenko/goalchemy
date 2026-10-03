"""lib.crypto.aes256gcm_encrypt: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_aes256gcm_encrypt(t, *args):
    crypto_call(t, 'aes256gcm_encrypt', args, 'bytes')
