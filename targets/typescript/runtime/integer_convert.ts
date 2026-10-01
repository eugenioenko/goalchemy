// core.integer.convert: truncate to the target width and reinterpret signedness.

export function to_i8(a: number | bigint): number {
  if (typeof a === "bigint") return Number(BigInt.asIntN(8, a));
  const x = a;
  return (x << 24) >> 24;
}
export function to_i16(a: number | bigint): number {
  if (typeof a === "bigint") return Number(BigInt.asIntN(16, a));
  const x = a;
  return (x << 16) >> 16;
}
export function to_i32(a: number | bigint): number {
  if (typeof a === "bigint") return Number(BigInt.asIntN(32, a));
  const x = a;
  return x | 0;
}
export function to_i64(a: number | bigint): bigint {
  return BigInt.asIntN(64, typeof a === "bigint" ? a : BigInt(a));
}
export function to_u8(a: number | bigint): number {
  if (typeof a === "bigint") return Number(BigInt.asUintN(8, a));
  const x = a;
  return x & 0xff;
}
export function to_u16(a: number | bigint): number {
  if (typeof a === "bigint") return Number(BigInt.asUintN(16, a));
  const x = a;
  return x & 0xffff;
}
export function to_u32(a: number | bigint): number {
  if (typeof a === "bigint") return Number(BigInt.asUintN(32, a));
  const x = a;
  return x >>> 0;
}
export function to_u64(a: number | bigint): bigint {
  return BigInt.asUintN(64, typeof a === "bigint" ? a : BigInt(a));
}
