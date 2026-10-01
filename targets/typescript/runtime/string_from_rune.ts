// core.string.from_rune: string(x) for an integer, encoding one code point.
import { encode } from "../types/utf8.ts";

export function fromRune(x: number | bigint): string {
  if (typeof x === "bigint") {
    if (x < 0n || x > 0x10ffffn) return encode(0xfffd);
    return encode(Number(x));
  }
  return encode(x);
}
