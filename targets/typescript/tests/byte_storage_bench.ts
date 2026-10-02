// Diagnostic evidence, never a noisy RSS pass/fail test. Run with --expose-gc.
import assert from "node:assert/strict";
import { performance } from "node:perf_hooks";
import { makeSlice } from "../runtime/slice_make.ts";
import { copy } from "../runtime/slice_copy.ts";
import { appendSlice } from "../runtime/slice_append.ts";

const bytes = process.argv[2] === "typed";
const n = Number(process.argv[3]) * 1024 * 1024;
const gc = (globalThis as any).gc as () => void;
assert.equal(typeof gc, "function");
gc();
const before = process.memoryUsage();
const begin = performance.now();
const src = makeSlice(n, n, () => 0, bytes);
const dst = makeSlice(n, n, () => 0, bytes);
for (let i = 0; i < n; i++) src.a![i] = i & 255;
const allocMs = performance.now() - begin;
const copyStart = performance.now();
for (let i = 0; i < 16; i++) copy(dst, src);
const copyMs = performance.now() - copyStart;
const appendStart = performance.now();
const grown = appendSlice(src, src);
const appendMs = performance.now() - appendStart;
gc();
const after = process.memoryUsage();
assert.equal(src.a instanceof Uint8Array, bytes);
assert.equal(dst.a instanceof Uint8Array, bytes);
assert.equal(grown.a instanceof Uint8Array, bytes);
assert.equal(grown.a![n + n - 1], 255);
assert.equal(dst.a![n - 1], 255);
console.log(JSON.stringify({node:process.version, mode:bytes ? "typed" : "boxed", MiB:n/1048576,
 liveElements:src.c+dst.c+grown.c, backingBytes:bytes ? (src.a as Uint8Array).byteLength+(dst.a as Uint8Array).byteLength+(grown.a as Uint8Array).byteLength : null,
 heapDelta:after.heapUsed-before.heapUsed, arrayBufferDelta:after.arrayBuffers-before.arrayBuffers,
 rssDelta:after.rss-before.rss, allocMs, copyMs, copyMiBPerSec:16*n/1048576/(copyMs/1000), appendMs}));
