// Representation checks complement the independently evaluated Go fixture.
import assert from "node:assert/strict";
import test from "node:test";
import { BYTE_NIL, NIL, Slice, fromArray, toArray } from "../types/slice.ts";
import { makeSlice } from "../runtime/slice_make.ts";
import { append, appendSlice, appendString } from "../runtime/slice_append.ts";
import { copy } from "../runtime/slice_copy.ts";
import { clearSlice } from "../runtime/slice_clear.ts";
import { reslice, sliceArray } from "../runtime/slice_slice.ts";
import { sliceToArray } from "../runtime/slice_to_array.ts";
import { toBytes } from "../runtime/string_to_bytes.ts";
import { fromBytes } from "../runtime/string_from_bytes.ts";

import { decSlice, view } from "./harness/codec.ts";

function native(b: Slice<number>): Uint8Array {
  assert.ok(b.a instanceof Uint8Array);
  assert.equal(b.a.byteLength, b.c + b.o);
  return b.a;
}

test("byte construction and nil append retain native storage", () => {
  assert.equal(append(BYTE_NIL, []), BYTE_NIL);
  assert.equal(appendSlice(BYTE_NIL, BYTE_NIL), BYTE_NIL);
  assert.equal(reslice(BYTE_NIL, 0, 0, 0), BYTE_NIL);
  assert.equal(BYTE_NIL.a, null);
  native(append(reslice(BYTE_NIL, 0), [255, 128, 0]));
  native(makeSlice(0, 0, () => 0, true));
  const b = makeSlice(2, 8, () => 0, true);
  assert.deepEqual([...native(b)], [0,0,0,0,0,0,0,0]);
  const a = native(b);
  const grown = append(b, [1,2,3,4,5,6,7]);
  assert.notEqual(native(grown), a);
  assert.equal(grown.c, 16);
  assert.deepEqual([...native(grown).subarray(9)], [0,0,0,0,0,0,0]);
  const decoded = decSlice({slice:["255"], cap:"3"}, Number, true);
  assert.deepEqual([...native(decoded)], [255,0,0]);
  native(append(view(BYTE_NIL, {}), [1]));
  const host = new Uint8Array([1,2,3]);
  assert.equal(fromArray(host).a, host);
  assert.equal(sliceArray(host, 1).a, host);
  const array = sliceToArray(sliceArray(host), 2);
  assert.ok(array instanceof Uint8Array);
  array[0] = 9;
  assert.equal(host[0], 1);
  assert.ok(sliceToArray(BYTE_NIL, 0) instanceof Uint8Array);
});

test("typed copy/append/clear retain aliased buffers", () => {
  const b = toBytes("\x01\x02\x03\x04\x05\x06");
  const a = native(b);
  copy(reslice(b, 1), b);
  assert.deepEqual([...a], [1,1,2,3,4,5]);
  copy(b, reslice(b, 2));
  assert.deepEqual([...a], [2,3,4,5,4,5]);
  const appended = appendSlice(reslice(b, 0, 2), reslice(b, 1, 5));
  assert.equal(appended.a, a);
  assert.deepEqual([...a], [2,3,3,4,5,4]);
  clearSlice(reslice(b, 2, 4), () => 0);
  assert.deepEqual([...a], [2,3,0,0,5,4]);
  const detached = appendSlice(reslice(b, 1, 3, 3), b);
  assert.notEqual(native(detached), a);
  assert.deepEqual(toArray(detached), [3,0,2,3,0,0,5,4]);
});

test("arbitrary byte strings copy independently; other elements stay generic", () => {
  const s = "\x00\xff\x80\xc0\xafA";
  const b = toBytes(s);
  const a = native(b);
  assert.equal(fromBytes(b), s);
  a[0] = 100;
  assert.equal(s.charCodeAt(0), 0);
  const out = appendString(BYTE_NIL, s);
  assert.equal(fromBytes(out), s);
  assert.ok(toBytes("").a instanceof Uint8Array);
  const generic = append(NIL, [1000, -1]);
  assert.ok(Array.isArray(generic.a));
  assert.deepEqual(toArray(generic), [1000, -1]);
  assert.ok(Array.isArray(makeSlice(2, 3, () => 0n).a));
});
