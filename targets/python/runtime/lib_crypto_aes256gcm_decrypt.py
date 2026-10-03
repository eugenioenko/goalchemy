"""lib.crypto.aes256gcm_decrypt: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_aes256gcm_decrypt(t, *args):
    crypto_call(t, 'aes256gcm_decrypt', args, 'bytes')
