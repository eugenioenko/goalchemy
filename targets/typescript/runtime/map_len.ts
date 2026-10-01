// core.map.len: number of entries.
import type { GoMap } from "../types/map.ts";

export function mapLen<K, V>(m: GoMap<K, V> | null): bigint {
  return m === null ? 0n : BigInt(m.index.size);
}
