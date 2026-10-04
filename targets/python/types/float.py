"""Explicit IEEE width rounding, conversions and nonreflexive map keys."""
import math
import struct


def round_float(x, bits):
    if bits == 64:
        return float(x)
    try:
        return struct.unpack('=f', struct.pack('=f', x))[0]
    except OverflowError:
        return math.copysign(math.inf, x)


def integer_float(value, bits):
    negative = value < 0
    n = -value if negative else value
    if not n:
        return 0.0
    shift = max(0, n.bit_length() - (24 if bits == 32 else 53))
    significand = n >> shift
    if shift:
        remainder = n - (significand << shift)
        half = 1 << (shift - 1)
        if remainder > half or (remainder == half and significand & 1):
            significand += 1
    result = math.ldexp(float(significand), shift)
    return -result if negative else result


def float_integer(x, bits, signed):
    if not signed and bits == 64:
        n = math.trunc(x) if math.isfinite(x) and -(2**63) <= x < 2**64 else 1 << 63
    else:
        width = 32 if bits <= 16 or (bits == 32 and signed) else 64
        limit = 2**(width - 1)
        n = math.trunc(x) if math.isfinite(x) and -limit <= x < limit else -limit
    n &= (1 << bits) - 1
    return n - (1 << bits) if signed and n >= 1 << (bits - 1) else n


def float_key(x):
    return object() if math.isnan(x) else (x if x else 0.0)


def float_div(a, b):
    if b:
        return a / b
    if a == 0 or math.isnan(a):
        return math.nan
    return math.copysign(math.inf, math.copysign(1.0, a) * math.copysign(1.0, b))


def float_min(a, b):
    if math.isnan(a) or math.isnan(b):
        return math.nan
    if a == b == 0:
        return -0.0 if math.copysign(1.0, a) < 0 or math.copysign(1.0, b) < 0 else 0.0
    return a if a < b else b


def float_max(a, b):
    if math.isnan(a) or math.isnan(b):
        return math.nan
    if a == b == 0:
        return 0.0 if math.copysign(1.0, a) > 0 or math.copysign(1.0, b) > 0 else -0.0
    return a if a > b else b


def float_print(x, bits):
    """Go1.27 shortest width-aware print; host formatting rounds decimal candidates."""
    x = round_float(x, bits)
    if math.isnan(x):
        return b'NaN'
    if math.isinf(x):
        return b'-Inf' if x < 0 else b'+Inf'
    if x == 0:
        return b'-0' if math.copysign(1.0, x) < 0 else b'0'
    sign = '-' if x < 0 else ''
    x = abs(x)
    for n in range(1, (9 if bits == 32 else 17) + 1):
        text = format(x, '.' + str(n - 1) + 'e')
        if round_float(float(text), bits) == x:
            break
    mantissa, exponent = text.split('e')
    digits, exp = mantissa.replace('.', '').rstrip('0'), int(exponent)
    if exp < -4 or exp >= 6:
        result = digits[0] + ('.' + digits[1:] if len(digits) > 1 else '') + 'e' + ('-' if exp < 0 else '+') + str(abs(exp)).zfill(2)
    else:
        point = exp + 1
        result = ('0.' + '0' * -point + digits) if point <= 0 else (digits + '0' * (point - len(digits))) if point >= len(digits) else digits[:point] + '.' + digits[point:]
    return (sign + result).encode('ascii')
