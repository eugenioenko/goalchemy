// core.integer.and: bitwise AND.
import { wrapU32, wrapU64 } from "../types/integer.ts";

export function and_i8(a: number, b: number): number {
  return a & b;
}
export function and_i16(a: number, b: number): number {
  return a & b;
}
export function and_i32(a: number, b: number): number {
  return a & b;
}
export function and_i64(a: bigint, b: bigint): bigint {
  return a & b;
}
export function and_u8(a: number, b: number): number {
  return a & b;
}
export function and_u16(a: number, b: number): number {
  return a & b;
}
export function and_u32(a: number, b: number): number {
  return wrapU32(a & b);
}
export function and_u64(a: bigint, b: bigint): bigint {
  return wrapU64(a & b);
}
