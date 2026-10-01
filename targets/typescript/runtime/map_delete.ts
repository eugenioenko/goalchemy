// core.map.delete: delete(m, k); a nil map is a no-op after hashing the key.
import type { GoMap } from "../types/map.ts";

export function mapDelete<K, V>(m: GoMap<K, V> | null, k: K, keyOf: (k: K) => unknown): void {
  const key = keyOf(k);
  if (m === null) return;
  const e = m.index.get(key);
  if (e !== undefined) {
    e.live = false;
    m.index.delete(key);
    m.compact();
  }
}
