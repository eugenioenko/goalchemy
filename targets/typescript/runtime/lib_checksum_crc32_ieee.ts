import type { Slice } from "../types/slice.ts";
import { nativeCRC32IEEE } from "../types/host.ts";

// Portable slicing-by-8 fallback for browsers, which have no standard CRC API.
const tables = new Uint32Array(8 * 256);
for (let i = 0; i < 256; i++) {
  let c = i;
  for (let bit = 0; bit < 8; bit++) c = (c >>> 1) ^ ((c & 1) ? 0xedb88320 : 0);
  tables[i] = c >>> 0;
}
for (let row = 1; row < 8; row++) {
  for (let i = 0; i < 256; i++) {
    const c = tables[(row - 1) * 256 + i];
    tables[row * 256 + i] = ((c >>> 8) ^ tables[c & 255]) >>> 0;
  }
}

/** IEEE CRC-32, preserving Go uint32's unsigned number representation. */
export function libChecksumCRC32IEEE(data: Slice<number>): number {
  if (nativeCRC32IEEE) {
    const backing = data.a;
    const bytes = backing instanceof Uint8Array
      ? backing.subarray(data.o, data.o + data.l)
      : Uint8Array.from({ length: data.l }, (_, i) => backing![data.o + i]);
    return nativeCRC32IEEE(bytes) >>> 0;
  }
  let crc = 0xffffffff;
  const bytes = data.a;
  let i = data.o;
  const end = i + data.l;
  for (; i + 8 <= end; i += 8) {
    crc ^= bytes![i] | (bytes![i + 1] << 8) | (bytes![i + 2] << 16) | (bytes![i + 3] << 24);
    crc = tables[1792 + (crc & 255)] ^ tables[1536 + ((crc >>> 8) & 255)] ^
      tables[1280 + ((crc >>> 16) & 255)] ^ tables[1024 + (crc >>> 24)] ^
      tables[768 + bytes![i + 4]] ^ tables[512 + bytes![i + 5]] ^
      tables[256 + bytes![i + 6]] ^ tables[bytes![i + 7]];
  }
  for (; i < end; i++) crc = (crc >>> 8) ^ tables[(crc ^ bytes![i]) & 255];
  return (~crc) >>> 0;
}
