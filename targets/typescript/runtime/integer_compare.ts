// core.integer.compare: three-way comparison.
export function compare_int(a: number | bigint, b: number | bigint): bigint {
  return a < b ? -1n : a > b ? 1n : 0n;
}
