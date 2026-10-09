from ..types.log import log_emit

def lib_log_emit(level,unix_nano,message,attrs,text):log_emit(level,unix_nano,message,[] if attrs.a is None else attrs.a[attrs.o:attrs.o+attrs.l],text)
