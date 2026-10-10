import os
from .lib_os_read_file import MAX_FILE, os_path, os_status

def lib_os_write_file(name,data,perm):
    bad=os_path(name)
    if bad is not None:return bad
    if data.l>MAX_FILE:return 6
    body=b'' if data.a is None else bytes(data.a[data.o:data.o+data.l])
    try:
        fd=os.open(name,os.O_WRONLY|os.O_CREAT|os.O_TRUNC|getattr(os,'O_CLOEXEC',0),perm&0o777)
        try:
            view=memoryview(body)
            while view:view=view[os.write(fd,view):]
        finally:os.close(fd)
    except OSError as e:return os_status(e)
    return 0
