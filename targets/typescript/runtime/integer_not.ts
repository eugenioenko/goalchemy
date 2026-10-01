// core.integer.not: bitwise complement.
import { wrapU8, wrapU16, wrapU32, wrapU64 } from "../types/integer.ts";

export function not_i8(a: number): number {
  return ~a;
}
export function not_i16(a: number): number {
  return ~a;
}
export function not_i32(a: number): number {
  return ~a;
}
export function not_i64(a: bigint): bigint {
  return ~a;
}
export function not_u8(a: number): number {
  return wrapU8(~a);
}
export function not_u16(a: number): number {
  return wrapU16(~a);
}
export function not_u32(a: number): number {
  return wrapU32(~a);
}
export function not_u64(a: bigint): bigint {
  return wrapU64(~a);
}
