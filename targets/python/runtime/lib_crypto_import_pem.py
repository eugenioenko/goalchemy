"""lib.crypto.import_pem: maintained native implementation."""
from .lib_crypto_close import crypto_call

def lib_crypto_import_pem(t, *args):
    crypto_call(t, 'import_pem', args, 'key')
