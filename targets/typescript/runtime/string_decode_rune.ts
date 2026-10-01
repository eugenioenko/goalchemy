// core.string.decode_rune: one rune at a byte offset, as range does.
import { decode } from "../types/utf8.ts";

export function decodeRune(s: string, i: number | bigint): [number, bigint] {
  const [r, w] = decode(s, Number(i));
  return [r, BigInt(w)];
}
