// Interface values, type descriptors, and dynamic dispatch.

export type Method = (recv: any, ...args: any[]) => any;

export interface TypeDesc {
  id: number;
  name: string;
  kind: string;
  /** Equality of two values of this type; throws if uncomparable. */
  eq: (a: any, b: any) => boolean;
  /** Map key encoding; throws if unhashable. */
  key: (a: any) => unknown;
  /** Method table keyed by Go method identity. */
  methods: Record<string, Method>;
  /** False when values of the type are not comparable. */
  comparable?: boolean;
  /** Format for uncaught panic printing, if the type is a named basic type. */
  basic?: "int" | "bool" | "string" | "float32" | "float64";
}

let nextTypeId = 1;

export function typeDesc(d: Omit<TypeDesc, "id"> & { id?: number }): TypeDesc {
  return { ...d, id: d.id ?? nextTypeId++ } as TypeDesc;
}

/** A non-nil interface value: dynamic type and value. */
export interface Box {
  t: TypeDesc;
  v: any;
}

export function box(t: TypeDesc, v: any): Box {
  return { t, v };
}

export function ifaceEq(a: Box | null, b: Box | null): boolean {
  if (a === null || b === null) return a === b;
  if (a.t !== b.t) return false;
  return a.t.eq(a.v, b.v);
}

export function ifaceKey(a: Box | null): unknown {
  if (a === null) return "nil";
  return a.t.id + ":" + String(a.t.key(a.v));
}

/** Reports whether t has every method in ids. */
export function implementsAll(t: TypeDesc, ids: readonly string[]): string | null {
  for (const id of ids) {
    if (!(id in t.methods)) return id;
  }
  return null;
}

let objIds = new WeakMap<object, number>();
let nextObjId = 1;

/** Stable identity number for reference-keyed map encodings. */
export function objId(o: object | null): number {
  if (o === null) return 0;
  let id = objIds.get(o);
  if (id === undefined) {
    id = nextObjId++;
    objIds.set(o, id);
  }
  return id;
}
