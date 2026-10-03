"""IEEE CRC-32 using the standard library's native bulk implementation."""
import zlib


def lib_checksum_crc32_ieee(data):
    if data.l == 0:
        return 0
    if isinstance(data.a, (bytes, bytearray, memoryview)):
        view = memoryview(data.a)[data.o:data.o + data.l]
    else:
        view = bytes(data.a[data.o:data.o + data.l])
    return zlib.crc32(view) & 0xffffffff
