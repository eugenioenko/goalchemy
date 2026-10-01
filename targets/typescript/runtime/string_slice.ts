// core.string.slice: substring by byte bounds.
import { check2 } from "../types/bounds.ts";

export function sslice(s: string, lo?: number | bigint, hi?: number | bigint): string {
  const l = lo ?? 0;
  const h = hi ?? s.length;
  check2(l, h, s.length, "length");
  return s.substring(Number(l), Number(h));
}
