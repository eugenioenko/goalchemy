// core.slice.store: write s[i] = v into shared backing storage.
import type { Slice } from "../types/slice.ts";
import { idx } from "../types/panic.ts";

export function sset<T>(s: Slice<T>, i: number | bigint, v: T): void {
  s.a![s.o + idx(i, s.l)] = v;
}
