// core.string.from_bytes: string(b), copying the bytes.
import type { Slice } from "../types/slice.ts";

export function fromBytes(b: Slice<number>): string {
  if (b.a === null) return "";
  let out = "";
  for (let i = 0; i < b.l; i += 8192) {
    out += String.fromCharCode(...b.a.slice(b.o + i, b.o + Math.min(b.l, i + 8192)));
  }
  return out;
}
