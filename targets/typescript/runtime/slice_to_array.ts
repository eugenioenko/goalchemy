// core.slice.to_array: [N]T(s), copying the first N elements.
import type { Slice } from "../types/slice.ts";
import { runtimePanic } from "../types/panic.ts";

export function sliceToArray<T>(s: Slice<T>, n: number, clone?: (v: T) => T): T[] {
  if (s.l < n) {
    throw runtimePanic(`cannot convert slice with length ${s.l} to array or pointer to array with length ${n}`);
  }
  const a = new Array<T>(n);
  for (let i = 0; i < n; i++) {
    const v = s.a![s.o + i];
    a[i] = clone ? clone(v) : v;
  }
  return a;
}
