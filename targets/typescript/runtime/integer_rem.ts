// core.integer.rem: remainder with the sign of the dividend; a zero divisor panics.
import { divZero } from "../types/integer.ts";

export function rem_i8(a: number, b: number): number {
  if (b === 0) divZero();
  return (a % b) + 0;
}
export function rem_i16(a: number, b: number): number {
  if (b === 0) divZero();
  return (a % b) + 0;
}
export function rem_i32(a: number, b: number): number {
  if (b === 0) divZero();
  return (a % b) + 0;
}
export function rem_i64(a: bigint, b: bigint): bigint {
  if (b === 0n) divZero();
  return a % b;
}
export function rem_u8(a: number, b: number): number {
  if (b === 0) divZero();
  return (a % b) + 0;
}
export function rem_u16(a: number, b: number): number {
  if (b === 0) divZero();
  return (a % b) + 0;
}
export function rem_u32(a: number, b: number): number {
  if (b === 0) divZero();
  return (a % b) + 0;
}
export function rem_u64(a: bigint, b: bigint): bigint {
  if (b === 0n) divZero();
  return a % b;
}
