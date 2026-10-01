// core.integer.shr: arithmetic right shift for signed kinds, logical for unsigned.
import { shiftCount } from "../types/integer.ts";

export function shr_i8(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 8) return a < 0 ? -1 : 0;
  return a >> n;
}
export function shr_i16(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 16) return a < 0 ? -1 : 0;
  return a >> n;
}
export function shr_i32(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 32) return a < 0 ? -1 : 0;
  return a >> n;
}
export function shr_i64(a: bigint, c: number | bigint): bigint {
  const n = shiftCount(c);
  if (n >= 64) return a < 0n ? -1n : 0n;
  return a >> BigInt(n);
}
export function shr_u8(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 8) return 0;
  return a >>> n;
}
export function shr_u16(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 16) return 0;
  return a >>> n;
}
export function shr_u32(a: number, c: number | bigint): number {
  const n = shiftCount(c);
  if (n >= 32) return 0;
  return a >>> n;
}
export function shr_u64(a: bigint, c: number | bigint): bigint {
  const n = shiftCount(c);
  if (n >= 64) return 0n;
  return a >> BigInt(n);
}
