//! IEEE CRC-32 through crc32fast, with runtime-selected CPU acceleration.
use super::*;

pub fn lib_checksum_crc32_ieee(data: V) -> V {
    let (h, offset, len, _, _) = slice_parts(&data);
    if len == 0 {
        return V::Int(0);
    }
    let crc = with(h, |obj| match obj {
        Obj::Bytes(bytes) => crc32fast::hash(&bytes[offset as usize..(offset + len) as usize]),
        _ => fault("byte slice backing expected"),
    });
    V::Int(crc as i64)
}
