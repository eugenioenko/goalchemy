// core.map.clear: remove every entry.
import type { GoMap } from "../types/map.ts";

export function mapClear<K, V>(m: GoMap<K, V> | null): void {
  if (m === null) return;
  for (const e of m.entries) e.live = false;
  m.index.clear();
  m.entries = [];
}
