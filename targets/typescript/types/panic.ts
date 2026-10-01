// Panic and runtime error representation for the TypeScript target.
import { box, type Box, type TypeDesc, typeDesc } from "./iface.ts";

/** A source panic carrying a Go interface value. */
export class GoPanic {
  value: Box | null;
  recovered = false;
  prev: GoPanic | null = null;
  constructor(value: Box | null) {
    this.value = value;
  }
}

/** An implementation fault: never a source panic. */
export class Fault extends Error {}

export function fault(msg: string): Fault {
  return new Fault("goalchemy fault: " + msg);
}

function errorType(name: string, prefix: string): TypeDesc {
  return typeDesc({
    name,
    kind: "runtime_error",
    eq: (a: unknown, b: unknown) => a === b,
    key: (a: unknown) => String(a),
    methods: {
      Error: (msg: string) => prefix + msg,
      RuntimeError: () => {},
    },
  });
}

/** runtime.Error values with the "runtime error: " prefix. */
export const RUNTIME_ERROR = errorType("runtime.Error", "runtime error: ");
/** runtime.Error values without the prefix, such as nil map writes. */
export const PLAIN_ERROR = errorType("runtime.plainError", "");
/** *runtime.TypeAssertionError. */
export const TYPE_ASSERTION_ERROR = errorType("*runtime.TypeAssertionError", "");

export function runtimePanic(msg: string): GoPanic {
  return new GoPanic(box(RUNTIME_ERROR, msg));
}

export function plainPanic(msg: string): GoPanic {
  return new GoPanic(box(PLAIN_ERROR, msg));
}

export function uncomparable(name: string): GoPanic {
  return runtimePanic("comparing uncomparable type " + name);
}

export function unhashable(name: string): GoPanic {
  return runtimePanic("hash of unhashable type " + name);
}

export function nilDeref(): GoPanic {
  return runtimePanic("invalid memory address or nil pointer dereference");
}

export function nilchk<T>(p: T | null | undefined): T {
  if (p === null || p === undefined) throw nilDeref();
  return p;
}

/** Formats an integer operand for bounds messages. */
function n(x: number | bigint): string {
  return String(x);
}

export function indexPanic(i: number | bigint, len: number): GoPanic {
  if (i < 0) return runtimePanic(`index out of range [${n(i)}]`);
  return runtimePanic(`index out of range [${n(i)}] with length ${len}`);
}

/** Converts a checked index to a host index in [0, len). */
export function idx(i: number | bigint, len: number): number {
  if (typeof i === "bigint") {
    if (i < 0n || i >= BigInt(len)) throw indexPanic(i, len);
    return Number(i);
  }
  if (i < 0 || i >= len) throw indexPanic(i, len);
  return i;
}

export function toNumber(i: number | bigint): number {
  return typeof i === "bigint" ? Number(i) : i;
}

export function uncomparableEq(name: string): never {
  throw uncomparable(name);
}

export function unhashableKey(name: string): never {
  throw unhashable(name);
}
