import base64
import binascii
import re
from .lib_crypto_close import MAX, slice_bytes
from .std_errors_new import std_errors_new
from ..types.slice import BYTE_NIL

def lib_encoding_base64_decode(v):
    try:
        if type(v) is not bytes or len(v)>((MAX+2)//3)*4:raise ValueError()
        if re.fullmatch(rb'(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?',v) is None:raise ValueError()
        out=base64.b64decode(v,validate=True)
        if len(out)>MAX or base64.b64encode(out)!=v:raise ValueError()
        return slice_bytes(out),None
    except (ValueError,binascii.Error):return BYTE_NIL,std_errors_new(b'encoding: invalid input or size')
