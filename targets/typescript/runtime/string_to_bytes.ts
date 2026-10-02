// core.string.to_bytes: []byte(s) with capacity equal to length.
import { Slice } from "../types/slice.ts";

export function toBytes(s: string): Slice<number> {
  const a = new Uint8Array(s.length);
  for (let i = 0; i < s.length; i++) a[i] = s.charCodeAt(i);
  return new Slice(a, 0, a.length, a.length);
}
