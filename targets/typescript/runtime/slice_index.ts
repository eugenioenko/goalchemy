// core.slice.index: read s[i], checking the length.
import type { Slice } from "../types/slice.ts";
import { idx } from "../types/panic.ts";

export function sget<T>(s: Slice<T>, i: number | bigint): T {
  return s.a![s.o + idx(i, s.l)];
}
