import { BYTE_NIL, Slice } from "../types/slice.ts";
import { libChecksumCRC32IEEE } from "../runtime/lib_checksum_crc32_ieee.ts";

// Independent bit-at-a-time oracle; deliberately does not share lookup tables.
function oracle(bytes: Uint8Array): number {
  let crc = 0xffffffff;
  for (const byte of bytes) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ ((crc & 1) ? 0xedb88320 : 0);
  }
  return (~crc) >>> 0;
}

export function runCRC32Suite(): number {
  if (libChecksumCRC32IEEE(BYTE_NIL) !== 0) throw new Error("nil CRC");
  const check = new Uint8Array([49,50,51,52,53,54,55,56,57]);
  if (libChecksumCRC32IEEE(new Slice(check,0,9,9)) !== 0xcbf43926)
    throw new Error("known IEEE check vector or unsigned result");
  let state = 0x8badf00d;
  const sizes = [0,1,2,3,7,8,9,15,16,17,31,32,33,63,64,65,255,256,257,4095,4096,4097,65535,65536,65537,1048576];
  for (let i = 0; i < 32; i++) {
    state = (Math.imul(state,1664525) + 1013904223) >>> 0;
    sizes.push(state % 65537);
  }
  for (const size of sizes) {
    const backing = new Uint8Array(size + 22);
    for (let i = 0; i < backing.length; i++) {
      state = (Math.imul(state,1664525) + 1013904223) >>> 0;
      backing[i] = state >>> 24;
    }
    const before = backing.slice();
    const actual = libChecksumCRC32IEEE(new Slice(backing,11,size,size+11));
    const expected = oracle(backing.subarray(11,11+size));
    if (actual !== expected || actual < 0 || actual > 0xffffffff)
      throw new Error(`CRC length ${size}: ${actual} != ${expected}`);
    if (backing.some((byte,i) => byte !== before[i])) throw new Error("input mutated");
  }
  return sizes.length + 2;
}
