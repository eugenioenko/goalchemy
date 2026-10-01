// core.integer.shl: left shift by any integer count; negative counts panic.
import { shiftCount, wrap8, wrap16, wrapU8, wrapU16, wrapU32, wrap64, wrapU64 } from "../types/integer.ts";

export function shl_i8(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 8) return 0;
  return wrap8(a << n);
}
export function shl_i16(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 16) return 0;
  return wrap16(a << n);
}
export function shl_i32(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 32) return 0;
  return (a << n) | 0;
}
export function shl_i64(a: bigint, c: number | bigint): bigint {
  const n = shiftCount(c);
  if (n >= 64) return 0n;
  return wrap64(a << BigInt(n));
}
export function shl_u8(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 8) return 0;
  return wrapU8(a << n);
}
export function shl_u16(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 16) return 0;
  return wrapU16(a << n);
}
export function shl_u32(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 32) return 0;
  return (a << n) >>> 0;
}
export function shl_u64(a: bigint, c: number | bigint): bigint {
  const n = shiftCount(c);
  if (n >= 64) return 0n;
  return wrapU64(a << BigInt(n));
}
