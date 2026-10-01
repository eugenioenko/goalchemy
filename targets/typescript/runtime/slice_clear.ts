// core.slice.clear: zero the elements below the length.
import type { Slice } from "../types/slice.ts";

export function clearSlice<T>(s: Slice<T>, zero: () => T): void {
  for (let i = 0; i < s.l; i++) s.a![s.o + i] = zero();
}
