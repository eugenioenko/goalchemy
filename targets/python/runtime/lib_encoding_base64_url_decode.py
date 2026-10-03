import base64
import binascii
import re
from .lib_crypto_close import MAX, slice_bytes
from .std_errors_new import std_errors_new
from ..types.slice import BYTE_NIL

def lib_encoding_base64_url_decode(v):
    try:
        if type(v) is not bytes or len(v)>((MAX+2)//3)*4 or len(v)%4==1 or re.fullmatch(rb'[A-Za-z0-9_-]*',v) is None:raise ValueError()
        out=base64.b64decode(v+b'='*((-len(v))%4),altchars=b'-_',validate=True)
        if len(out)>MAX or base64.urlsafe_b64encode(out).rstrip(b'=')!=v:raise ValueError()
        return slice_bytes(out),None
    except (ValueError,binascii.Error):return BYTE_NIL,std_errors_new(b'encoding: invalid input or size')
