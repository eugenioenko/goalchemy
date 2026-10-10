import errno
import os
from ..types.slice import Slice, BYTE_NIL

MAX_FILE=1<<30
STATUS={errno.ENOENT:1,errno.EEXIST:2,errno.EACCES:3,errno.EPERM:3,errno.EISDIR:4,errno.ENOTDIR:5,errno.EFBIG:6,errno.ENOSYS:7,errno.EOPNOTSUPP:7,errno.EINVAL:8}

def os_path(name):
    if len(name)==0:return 1
    if b'\0' in name:return 8
    return None

def os_status(e):return STATUS.get(e.errno,9)

def lib_os_read_file(name):
    bad=os_path(name)
    if bad is not None:return BYTE_NIL,bad
    try:
        with open(name,'rb') as f:
            if os.fstat(f.fileno()).st_size>MAX_FILE:return BYTE_NIL,6
            data=bytearray(f.read(MAX_FILE+1))
    except OSError as e:return BYTE_NIL,os_status(e)
    if len(data)>MAX_FILE:return BYTE_NIL,6
    return Slice(data,0,len(data),len(data),True),0
