// core.slice.copy: copy(dst, src) through a temporary for overlap safety.
import type { Slice } from "../types/slice.ts";

export function copy<T>(dst: Slice<T>, src: Slice<T>, clone?: (v: T) => T): bigint {
  const n = Math.min(dst.l, src.l);
  if (n === 0) return 0n;
  const tmp = src.a!.slice(src.o, src.o + n);
  for (let i = 0; i < n; i++) dst.a![dst.o + i] = clone ? clone(tmp[i]) : tmp[i];
  return BigInt(n);
}

export function copyString(dst: Slice<number>, s: string): bigint {
  const n = Math.min(dst.l, s.length);
  for (let i = 0; i < n; i++) dst.a![dst.o + i] = s.charCodeAt(i);
  return BigInt(n);
}
