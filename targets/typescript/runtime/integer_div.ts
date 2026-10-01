// core.integer.div: truncating division; a zero divisor panics.
import { divZero, wrap32, wrapU32, wrap8, wrap16, wrap64, wrapU64 } from "../types/integer.ts";

export function div_i8(a: number, b: number): number {
  if (b === 0) divZero();
  return wrap8(Math.trunc(a / b));
}
export function div_i16(a: number, b: number): number {
  if (b === 0) divZero();
  return wrap16(Math.trunc(a / b));
}
export function div_i32(a: number, b: number): number {
  if (b === 0) divZero();
  return wrap32(Math.trunc(a / b));
}
export function div_i64(a: bigint, b: bigint): bigint {
  if (b === 0n) divZero();
  return wrap64(a / b);
}
export function div_u8(a: number, b: number): number {
  if (b === 0) divZero();
  return Math.trunc(a / b);
}
export function div_u16(a: number, b: number): number {
  if (b === 0) divZero();
  return Math.trunc(a / b);
}
export function div_u32(a: number, b: number): number {
  if (b === 0) divZero();
  return wrapU32(Math.trunc(a / b));
}
export function div_u64(a: bigint, b: bigint): bigint {
  if (b === 0n) divZero();
  return wrapU64(a / b);
}
