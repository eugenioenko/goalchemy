// core.integer.mul: wrapping multiplication; Math.imul keeps 32-bit products exact.
import { wrap16, wrap32, wrap64, wrap8, wrapU16, wrapU32, wrapU64, wrapU8 } from "../types/integer.ts";

export function mul_i8(a: number, b: number): number {
  return wrap8(a * b);
}
export function mul_i16(a: number, b: number): number {
  return wrap16(a * b);
}
export function mul_i32(a: number, b: number): number {
  return Math.imul(a, b);
}
export function mul_i64(a: bigint, b: bigint): bigint {
  return wrap64(a * b);
}
export function mul_u8(a: number, b: number): number {
  return wrapU8(a * b);
}
export function mul_u16(a: number, b: number): number {
  return wrapU16(a * b);
}
export function mul_u32(a: number, b: number): number {
  return Math.imul(a, b) >>> 0;
}
export function mul_u64(a: bigint, b: bigint): bigint {
  return wrapU64(a * b);
}
