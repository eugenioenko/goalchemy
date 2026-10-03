"""lib.crypto.hmac_sha256: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_hmac_sha256(t, *args):
    crypto_call(t, 'hmac_sha256', args, 'bytes')
