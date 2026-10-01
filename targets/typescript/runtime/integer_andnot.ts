// core.integer.andnot: bit clear.
import { wrapU32, wrapU64 } from "../types/integer.ts";

export function andnot_i8(a: number, b: number): number {
  return a & ~b;
}
export function andnot_i16(a: number, b: number): number {
  return a & ~b;
}
export function andnot_i32(a: number, b: number): number {
  return a & ~b;
}
export function andnot_i64(a: bigint, b: bigint): bigint {
  return a & ~b;
}
export function andnot_u8(a: number, b: number): number {
  return a & ~b;
}
export function andnot_u16(a: number, b: number): number {
  return a & ~b;
}
export function andnot_u32(a: number, b: number): number {
  return wrapU32(a & ~b);
}
export function andnot_u64(a: bigint, b: bigint): bigint {
  return wrapU64(a & ~b);
}
