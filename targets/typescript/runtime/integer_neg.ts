// core.integer.neg: wrapping negation.
import { wrap8, wrap16, wrap32, wrapU8, wrapU16, wrapU32, wrap64, wrapU64 } from "../types/integer.ts";

export function neg_i8(a: number): number {
  return wrap8(-a);
}
export function neg_i16(a: number): number {
  return wrap16(-a);
}
export function neg_i32(a: number): number {
  return wrap32(-a);
}
export function neg_i64(a: bigint): bigint {
  return wrap64(-a);
}
export function neg_u8(a: number): number {
  return wrapU8(-a);
}
export function neg_u16(a: number): number {
  return wrapU16(-a);
}
export function neg_u32(a: number): number {
  return wrapU32(-a);
}
export function neg_u64(a: bigint): bigint {
  return wrapU64(-a);
}
