// core.string.compare: bytewise comparison; binary strings compare by byte.
export function scompare(a: string, b: string): bigint {
  return a < b ? -1n : a > b ? 1n : 0n;
}
