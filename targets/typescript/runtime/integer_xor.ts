// core.integer.xor: bitwise XOR.
import { wrapU32, wrapU64 } from "../types/integer.ts";

export function xor_i8(a: number, b: number): number {
  return a ^ b;
}
export function xor_i16(a: number, b: number): number {
  return a ^ b;
}
export function xor_i32(a: number, b: number): number {
  return a ^ b;
}
export function xor_i64(a: bigint, b: bigint): bigint {
  return a ^ b;
}
export function xor_u8(a: number, b: number): number {
  return a ^ b;
}
export function xor_u16(a: number, b: number): number {
  return a ^ b;
}
export function xor_u32(a: number, b: number): number {
  return wrapU32(a ^ b);
}
export function xor_u64(a: bigint, b: bigint): bigint {
  return wrapU64(a ^ b);
}
