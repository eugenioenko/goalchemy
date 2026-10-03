import test from "node:test";
import assert from "node:assert/strict";
import { runCRC32Suite } from "./checksum_suite.ts";

test("native IEEE CRC32 vectors, random slices, chunk edges and ownership", () => {
  assert.equal(runCRC32Suite(), 60);
});
