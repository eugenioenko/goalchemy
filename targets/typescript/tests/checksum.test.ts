import test from "node:test";
import assert from "node:assert/strict";
import zlib from "node:zlib";
import { syncBuiltinESMExports } from "node:module";
import { BYTE_NIL, Slice } from "../types/slice.ts";
import { libChecksumCRC32IEEE } from "../runtime/lib_checksum_crc32_ieee.ts";
import { runCRC32Suite } from "./checksum_suite.ts";

test("portable IEEE CRC32 vectors, random slices, chunk edges and ownership", () => {
  assert.equal(runCRC32Suite(), 60);
});

test("Node executable adapter delegates every checksum to node:zlib.crc32", async () => {
  const original = zlib.crc32;
  const calls: Uint8Array[] = [];
  let sentinel = false;
  zlib.crc32 = bytes => {
    assert.ok(bytes instanceof Uint8Array);
    calls.push(bytes);
    return sentinel ? 0xfedcba98 : original(bytes);
  };
  syncBuiltinESMExports();
  try {
    await import("../types/node_host.ts");
    assert.equal(runCRC32Suite(), 60);
    assert.equal(calls.length, 60, "nil and empty also delegate");
    const storage = new Uint8Array([7,8,9,10,11,12]);
    const view = storage.subarray(1,5);
    const data = new Slice(view,1,2,3);
    assert.equal(libChecksumCRC32IEEE(data), original(new Uint8Array([9,10])));
    assert.equal(calls.at(-1)!.buffer, storage.buffer);
    assert.equal(calls.at(-1)!.byteOffset, storage.byteOffset + 2);
    assert.deepEqual(Array.from(calls.at(-1)!), [9,10]);
    assert.deepEqual(Array.from(storage), [7,8,9,10,11,12]);
    // Legacy generic backing storage remains valid byte input.
    assert.equal(libChecksumCRC32IEEE(new Slice([0,255,128,1],1,2,3)), original(new Uint8Array([255,128])));
    sentinel = true;
    assert.equal(libChecksumCRC32IEEE(BYTE_NIL), 0xfedcba98,
      "a builtin sentinel proves this path cannot silently use the fallback");
  } finally {
    zlib.crc32 = original;
    syncBuiltinESMExports();
  }
});
