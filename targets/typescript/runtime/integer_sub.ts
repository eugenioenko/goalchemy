// core.integer.sub: wrapping subtraction.
import { wrap16, wrap32, wrap64, wrap8, wrapU16, wrapU32, wrapU64, wrapU8 } from "../types/integer.ts";

export function sub_i8(a: number, b: number): number {
  return wrap8(a - b);
}
export function sub_i16(a: number, b: number): number {
  return wrap16(a - b);
}
export function sub_i32(a: number, b: number): number {
  return wrap32(a - b);
}
export function sub_i64(a: bigint, b: bigint): bigint {
  return wrap64(a - b);
}
export function sub_u8(a: number, b: number): number {
  return wrapU8(a - b);
}
export function sub_u16(a: number, b: number): number {
  return wrapU16(a - b);
}
export function sub_u32(a: number, b: number): number {
  return wrapU32(a - b);
}
export function sub_u64(a: bigint, b: bigint): bigint {
  return wrapU64(a - b);
}
