// IEEE scalars. Binary32 values are represented by already-rounded numbers.
export function roundFloat(x: number, bits: number): number {
  return bits === 32 ? Math.fround(x) : x;
}

/** Direct integer-to-IEEE rounding: do not round BigInt through binary64 first. */
export function integerFloat(value: bigint | number, bits: number): number {
  let n = typeof value === 'bigint' ? value : BigInt(value);
  const negative = n < 0n;
  if (negative) n = -n;
  if (n === 0n) return 0;
  const precision = bits === 32 ? 24 : 53;
  const shift = Math.max(0, n.toString(2).length - precision);
  let significand = n >> BigInt(shift);
  if (shift > 0) {
    const remainder = n - (significand << BigInt(shift));
    const half = 1n << BigInt(shift - 1);
    if (remainder > half || (remainder === half && (significand & 1n) !== 0n)) significand++;
  }
  const result = Number(significand) * 2 ** shift;
  return negative ? -result : result;
}

/** Selected Go1.27 linux/amd64 conversion results, including exceptional inputs. */
export function floatInteger(x: number, bits: number, signed: boolean): bigint | number {
  let n: bigint;
  if (!signed && bits === 64) {
    n = Number.isFinite(x) && x >= -(2 ** 63) && x < 2 ** 64 ? BigInt(Math.trunc(x)) : 1n << 63n;
  } else {
    // Go uses a signed32 instruction for small kinds, signed64 for uint32/int64.
    const width = bits <= 16 || (bits === 32 && signed) ? 32 : 64;
    const limit = 2 ** (width - 1);
    n = Number.isFinite(x) && x >= -limit && x < limit ? BigInt(Math.trunc(x)) : -(1n << BigInt(width - 1));
  }
  n = signed ? BigInt.asIntN(bits, n) : BigInt.asUintN(bits, n);
  return bits === 64 ? n : Number(n);
}

let nanKey = 0n;
/** Every NaN key operation must miss, including nested/interface encodings. */
export function floatKey(x: number): string {
  return Number.isNaN(x) ? 'nan:' + (++nanKey) : 'float:' + String(x === 0 ? 0 : x);
}

export function floatMin(a: number, b: number): number { return Math.min(a, b); }
export function floatMax(a: number, b: number): number { return Math.max(a, b); }

/** Go1.27 shortest, width-aware builtin print. Search correctly rounded decimal
 * candidates until parsing returns the original IEEE value, then apply Go's g layout. */
export function floatPrint(x: number, bits: number): string {
  x = roundFloat(x, bits);
  if (Number.isNaN(x)) return "NaN";
  if (!Number.isFinite(x)) return x < 0 ? "-Inf" : "+Inf";
  if (x === 0) return Object.is(x, -0) ? "-0" : "0";
  const sign = x < 0 ? "-" : "";
  x = Math.abs(x);
  // ECMAScript toExponential resolves decimal ties upwards; Go uses ties-even.
  // Form exact binary rational candidates so halfway cases stay portable.
  const raw = new DataView(new ArrayBuffer(8));
  raw.setFloat64(0, x, false);
  const binary = raw.getBigUint64(0, false);
  const exponent = Number(binary >> 52n);
  const significand = (binary & ((1n << 52n) - 1n)) | (exponent === 0 ? 0n : 1n << 52n);
  const power = (exponent === 0 ? -1022 : exponent - 1023) - 52;
  const numerator = power >= 0 ? significand << BigInt(power) : significand;
  const denominator = power >= 0 ? 1n : 1n << BigInt(-power);
  let exp = Number(x.toExponential().split("e")[1]);
  const belowPower = (e: number): boolean => e >= 0 ? numerator < denominator * 10n ** BigInt(e) : numerator * 10n ** BigInt(-e) < denominator;
  while (belowPower(exp)) exp--;
  while (!belowPower(exp + 1)) exp++;
  let digits = "", decimalExp = exp;
  for (let n = 1; n <= (bits === 32 ? 9 : 17); n++) {
    const scale = exp - n + 1;
    const p = scale < 0 ? numerator * 10n ** BigInt(-scale) : numerator;
    const q = scale > 0 ? denominator * 10n ** BigInt(scale) : denominator;
    let rounded = p / q;
    const remainder = p % q;
    if (remainder * 2n > q || (remainder * 2n === q && (rounded & 1n) !== 0n)) rounded++;
    digits = rounded.toString();
    decimalExp = exp + digits.length - n;
    if (roundFloat(Number(digits + "e" + scale), bits) === x) break;
  }
  exp = decimalExp;
  digits = digits.replace(/0+$/, "");
  if (exp < -4 || exp >= 6) return sign + digits[0] + (digits.length > 1 ? "." + digits.slice(1) : "") + "e" + (exp < 0 ? "-" : "+") + String(Math.abs(exp)).padStart(2, "0");
  const point = exp + 1;
  return sign + (point <= 0 ? "0." + "0".repeat(-point) + digits : point >= digits.length ? digits + "0".repeat(point - digits.length) : digits.slice(0, point) + "." + digits.slice(point));
}
