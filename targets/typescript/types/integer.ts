// Integer kinds. Kinds up to 32 bits are JavaScript numbers; 64-bit kinds,
// including int and uint, are bigints.

export type Small = number;
export type Wide = bigint;
export type Int = number | bigint;

export function wrap8(x: number): number {
  return (x << 24) >> 24;
}
export function wrap16(x: number): number {
  return (x << 16) >> 16;
}
export function wrapU8(x: number): number {
  return x & 0xff;
}
export function wrapU16(x: number): number {
  return x & 0xffff;
}
export function wrap32(x: number): number {
  return x | 0;
}
export function wrapU32(x: number): number {
  return x >>> 0;
}
export function wrap64(x: bigint): bigint {
  return BigInt.asIntN(64, x);
}
export function wrapU64(x: bigint): bigint {
  return BigInt.asUintN(64, x);
}

import { runtimePanic } from "./panic.ts";

/**
 * Validates a shift count of any integer kind. Returns the count clamped to
 * 64, which shifts every bit out of every kind.
 */
export function shiftCount(n: number | bigint): number {
  if (n < 0) throw runtimePanic("negative shift amount");
  if (typeof n === "bigint") return n > 64n ? 64 : Number(n);
  return n > 64 ? 64 : n;
}

export function divZero(): never {
  throw runtimePanic("integer divide by zero");
}
