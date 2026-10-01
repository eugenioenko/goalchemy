// core.string.from_runes: string(r) for a rune slice.
import type { Slice } from "../types/slice.ts";
import { encode } from "../types/utf8.ts";

export function fromRunes(r: Slice<number>): string {
  let out = "";
  for (let i = 0; i < r.l; i++) out += encode(r.a![r.o + i]);
  return out;
}
