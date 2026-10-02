// core.slice.make: make([]T, len, cap) with zeroed capacity.
import { Slice, allocate } from "../types/slice.ts";
import { runtimePanic, fault } from "../types/panic.ts";

const HOST_LIMIT = 2 ** 32 - 1;

export function makeSlice<T>(len: number | bigint, cap: number | bigint, zero: () => T, bytes = false): Slice<T> {
  if (len < 0 || len > Number.MAX_SAFE_INTEGER) throw runtimePanic("makeslice: len out of range");
  if (cap < len || cap > Number.MAX_SAFE_INTEGER) throw runtimePanic("makeslice: cap out of range");
  const c = Number(cap);
  if (c > HOST_LIMIT) throw fault("allocation of " + c + " elements exceeds host limits");
  const a = allocate<T>(c, bytes);
  if (!bytes) for (let i = 0; i < c; i++) a[i] = zero();
  return new Slice(a, 0, Number(len), c);
}
