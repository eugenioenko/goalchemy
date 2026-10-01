// core.map.store: m[k] = v; storing into a nil map panics.
import { Entry, type GoMap } from "../types/map.ts";
import { plainPanic } from "../types/panic.ts";

export function mapSet<K, V>(m: GoMap<K, V> | null, k: K, v: V): void {
  if (m === null) throw plainPanic("assignment to entry in nil map");
  const key = m.keyOf(k);
  const e = m.index.get(key);
  if (e !== undefined) {
    e.v = v;
    return;
  }
  const n = new Entry(k, v);
  m.index.set(key, n);
  m.entries.push(n);
}
