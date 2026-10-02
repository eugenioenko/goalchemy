// core.slice.clear: zero the elements below the length.
import type { Slice } from "../types/slice.ts";

export function clearSlice<T>(s: Slice<T>, zero: () => T): void {
  if (s.a instanceof Uint8Array) {
    s.a.fill(0, s.o, s.o + s.l);
    return;
  }
  for (let i = 0; i < s.l; i++) s.a![s.o + i] = zero();
}
