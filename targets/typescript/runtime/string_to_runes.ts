// core.string.to_runes: []rune(s) with capacity equal to length.
import { Slice } from "../types/slice.ts";
import { decode } from "../types/utf8.ts";

export function toRunes(s: string): Slice<number> {
  const a: number[] = [];
  for (let i = 0; i < s.length; ) {
    const [r, w] = decode(s, i);
    a.push(r);
    i += w;
  }
  return new Slice(a, 0, a.length, a.length);
}
