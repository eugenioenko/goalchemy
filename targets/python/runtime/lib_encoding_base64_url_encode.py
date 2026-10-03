import base64
from .lib_crypto_close import byte_input, Reject
from .std_errors_new import std_errors_new

def lib_encoding_base64_url_encode(v):
    try:return base64.urlsafe_b64encode(byte_input(v)).rstrip(b'='),None
    except Reject:return b'',std_errors_new(b'encoding: invalid input or size')
