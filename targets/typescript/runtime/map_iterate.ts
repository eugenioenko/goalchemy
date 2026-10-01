// core.map.iterate: snapshot iteration in insertion order.
import { MapIter, type GoMap } from "../types/map.ts";
import { Slice } from "../types/slice.ts";

export function mapIter<K, V>(m: GoMap<K, V> | null): MapIter<K, V> {
  return new MapIter(m === null ? [] : m.entries.slice());
}

/** Advances it; on true, it.k and it.v hold the entry's key and current value. */
export function mapNext<K, V>(it: MapIter<K, V>): boolean {
  while (it.i < it.entries.length) {
    const e = it.entries[it.i++];
    if (e.live) {
      it.k = e.k;
      it.v = e.v;
      return true;
    }
  }
  return false;
}

/** The keys a full iteration visits. */
export function mapKeys<K, V>(m: GoMap<K, V> | null): Slice<K> {
  const it = mapIter(m);
  const keys: K[] = [];
  while (mapNext(it)) keys.push(it.k as K);
  return new Slice(keys, 0, keys.length, keys.length);
}
