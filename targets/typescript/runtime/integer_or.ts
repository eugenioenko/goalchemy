// core.integer.or: bitwise OR.
import { wrapU32, wrapU64 } from "../types/integer.ts";

export function or_i8(a: number, b: number): number {
  return a | b;
}
export function or_i16(a: number, b: number): number {
  return a | b;
}
export function or_i32(a: number, b: number): number {
  return a | b;
}
export function or_i64(a: bigint, b: bigint): bigint {
  return a | b;
}
export function or_u8(a: number, b: number): number {
  return a | b;
}
export function or_u16(a: number, b: number): number {
  return a | b;
}
export function or_u32(a: number, b: number): number {
  return wrapU32(a | b);
}
export function or_u64(a: bigint, b: bigint): bigint {
  return wrapU64(a | b);
}
