// core.map.lookup: v, ok := m[k]. Nil maps still hash the key.
import type { GoMap } from "../types/map.ts";

export function mapGet<K, V>(m: GoMap<K, V> | null, k: K, keyOf: (k: K) => unknown, zero: () => V): [V, boolean] {
  const key = keyOf(k);
  if (m === null) return [zero(), false];
  const e = m.index.get(key);
  return e === undefined ? [zero(), false] : [e.v, true];
}
