// core.string.index: byte at an index.
import { idx } from "../types/panic.ts";

export function sindex(s: string, i: number | bigint): number {
  return s.charCodeAt(idx(i, s.length));
}
