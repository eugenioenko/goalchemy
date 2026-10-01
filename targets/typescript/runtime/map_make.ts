// core.map.make: a new empty map.
import { GoMap } from "../types/map.ts";

export function makeMap<K, V>(keyOf: (k: K) => unknown): GoMap<K, V> {
  return new GoMap<K, V>(keyOf);
}
