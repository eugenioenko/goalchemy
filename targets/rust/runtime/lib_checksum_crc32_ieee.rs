//! IEEE CRC-32 over native byte storage. No additional crate dependency.
use super::*;

const fn crc32_ieee_tables() -> [[u32; 256]; 8] {
    let mut tables = [[0u32; 256]; 8];
    let mut i = 0;
    while i < 256 {
        let mut crc = i as u32;
        let mut bit = 0;
        while bit < 8 {
            crc = (crc >> 1) ^ if crc & 1 != 0 { 0xedb88320 } else { 0 };
            bit += 1;
        }
        tables[0][i] = crc;
        i += 1;
    }
    let mut row = 1;
    while row < 8 {
        i = 0;
        while i < 256 {
            let crc = tables[row - 1][i];
            tables[row][i] = (crc >> 8) ^ tables[0][(crc & 255) as usize];
            i += 1;
        }
        row += 1;
    }
    tables
}

const CRC32_IEEE_TABLES: [[u32; 256]; 8] = crc32_ieee_tables();

fn crc32_ieee_bytes(bytes: &[u8]) -> u32 {
    let table = &CRC32_IEEE_TABLES;
    let mut crc = 0xffffffffu32;
    let mut chunks = bytes.chunks_exact(8);
    for b in &mut chunks {
        crc ^= u32::from_le_bytes([b[0], b[1], b[2], b[3]]);
        crc = table[7][(crc & 255) as usize]
            ^ table[6][((crc >> 8) & 255) as usize]
            ^ table[5][((crc >> 16) & 255) as usize]
            ^ table[4][(crc >> 24) as usize]
            ^ table[3][b[4] as usize]
            ^ table[2][b[5] as usize]
            ^ table[1][b[6] as usize]
            ^ table[0][b[7] as usize];
    }
    for b in chunks.remainder() {
        crc = (crc >> 8) ^ table[0][((crc ^ *b as u32) & 255) as usize];
    }
    !crc
}

pub fn lib_checksum_crc32_ieee(data: V) -> V {
    let (h, offset, len, _, _) = slice_parts(&data);
    if len == 0 {
        return V::Int(0);
    }
    let crc = with(h, |obj| match obj {
        Obj::Bytes(bytes) => crc32_ieee_bytes(&bytes[offset as usize..(offset + len) as usize]),
        _ => fault("byte slice backing expected"),
    });
    V::Int(crc as i64)
}
