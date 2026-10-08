// SPDX-License-Identifier: Apache-2.0
import Foundation

enum GNumeric {
  static func normalize(_ value: UInt64, _ width: Int, _ signed: Bool) -> GValue {
    return .integer(width == 64 ? value : value & ((UInt64(1) << width) - 1), width, signed)
  }
  static func rounded(_ value: Double, _ width: Int) -> GValue {
    .float(width == 32 ? Double(Float(value)) : value, width)
  }
  static func unary(_ op: String, _ value: GValue) throws -> GValue {
    if op == "not" { return .bool(!value.boolValue) }
    if case .float(let number, let bits) = value { return rounded(-number, bits) }
    guard case .integer(let number, let width, let signed) = value else {
      throw GFault("unary representation")
    }
    return normalize(op == "neg" ? 0 &- number : ~number, width, signed)
  }
  static func binary(_ op: String, _ a: GValue, _ b: GValue) throws -> GValue {
    if op == "==" || op == "!=" {
      let equal = try GEqual(a, b)
      return .bool(op == "==" ? equal : !equal)
    }
    if case .string(let x) = a, case .string(let y) = b {
      switch op {
      case "+": return .string(x + y)
      case "<": return .bool(x.lexicographicallyPrecedes(y))
      case ">": return .bool(y.lexicographicallyPrecedes(x))
      case "<=": return .bool(!y.lexicographicallyPrecedes(x))
      case ">=": return .bool(!x.lexicographicallyPrecedes(y))
      case "min": return x.lexicographicallyPrecedes(y) ? a : b
      case "max": return y.lexicographicallyPrecedes(x) ? a : b
      default: throw GFault("string operation")
      }
    }
    if case .float(let x, let bits) = a, case .float(let y, _) = b {
      switch op {
      case "<": return .bool(x < y)
      case "<=": return .bool(x <= y)
      case ">": return .bool(x > y)
      case ">=": return .bool(x >= y)
      case "+": return rounded(x + y, bits)
      case "-": return rounded(x - y, bits)
      case "*": return rounded(x * y, bits)
      case "/": return rounded(x / y, bits)
      case "min", "max":
        if x.isNaN || y.isNaN { return rounded(Double.nan, bits) }
        if x == 0 && y == 0 {
          let negative =
            op == "min"
            ? (x.sign == .minus || y.sign == .minus) : (x.sign == .minus && y.sign == .minus)
          return rounded(negative ? -0.0 : 0.0, bits)
        }
        return op == "min" ? (x < y ? a : b) : (x > y ? a : b)
      default: throw GFault("float operation")
      }
    }
    guard case .integer(let x, let width, let signed) = a else {
      throw GFault("integer representation")
    }
    let y = b.unsigned
    switch op {
    case "<": return .bool(signed ? a.intValue < b.intValue : x < y)
    case "<=": return .bool(signed ? a.intValue <= b.intValue : x <= y)
    case ">": return .bool(signed ? a.intValue > b.intValue : x > y)
    case ">=": return .bool(signed ? a.intValue >= b.intValue : x >= y)
    case "min": return signed ? (a.intValue < b.intValue ? a : b) : (x < y ? a : b)
    case "max": return signed ? (a.intValue > b.intValue ? a : b) : (x > y ? a : b)
    case "+": return normalize(x &+ y, width, signed)
    case "-": return normalize(x &- y, width, signed)
    case "*": return normalize(x &* y, width, signed)
    case "&": return normalize(x & y, width, signed)
    case "|": return normalize(x | y, width, signed)
    case "^": return normalize(x ^ y, width, signed)
    case "&^": return normalize(x & ~y, width, signed)
    case "/", "%":
      if y == 0 { throw GPanic.runtime("integer divide by zero") }
      if signed {
        let numerator = a.intValue
        let denominator = b.intValue
        if numerator == Int64.min && denominator == -1 {
          return normalize(op == "/" ? x : 0, width, signed)
        }
        return normalize(
          UInt64(bitPattern: op == "/" ? numerator / denominator : numerator % denominator), width,
          signed)
      }
      return normalize(op == "/" ? x / y : x % y, width, signed)
    case "<<", ">>":
      if case .integer(_, _, true) = b, b.intValue < 0 {
        throw GPanic.runtime("negative shift amount")
      }
      if y >= UInt64(width) {
        return normalize(op == ">>" && signed && a.intValue < 0 ? ~UInt64(0) : 0, width, signed)
      }
      if op == "<<" { return normalize(x << Int(y), width, signed) }
      return normalize(
        signed ? UInt64(bitPattern: a.intValue >> Int(y)) : x >> Int(y), width, signed)
    default: throw GFault("integer operation " + op)
    }
  }
  static func convert(_ value: GValue, _ bits: Int, _ signed: Bool, _ floating: Bool) -> GValue {
    if floating {
      if case .float(let number, _) = value { return rounded(number, bits) }
      if bits == 32 {
        return .float(
          Double(signedInteger(value) ? Float(value.intValue) : Float(value.unsigned)), 32)
      }
      return .float(signedInteger(value) ? Double(value.intValue) : Double(value.unsigned), 64)
    }
    guard case .float(let x, _) = value else {
      return normalize(
        signedInteger(value) ? UInt64(bitPattern: value.intValue) : value.unsigned, bits, signed)
    }
    let number = x.rounded(.towardZero)
    if !signed && bits == 64 {
      if !number.isFinite || number < -9223372036854775808.0 || number >= 18446744073709551616.0 {
        return normalize(1 << 63, bits, signed)
      }
      return normalize(
        number < 0 ? UInt64(bitPattern: Int64(number)) : UInt64(number), bits, signed)
    }
    let width = (bits <= 16 || (bits == 32 && signed)) ? 32 : 64
    let limit = width == 32 ? 2147483648.0 : 9223372036854775808.0
    if !number.isFinite || number < -limit || number >= limit {
      return normalize(
        width == 32 ? UInt64(bitPattern: -2_147_483_648) : UInt64(1) << 63, bits, signed)
    }
    return normalize(UInt64(bitPattern: Int64(number)), bits, signed)
  }
  static func signedInteger(_ value: GValue) -> Bool {
    if case .integer(_, _, let signed) = value { return signed }
    return false
  }
  static func floatText(_ number: Double, _ bits: Int) -> String {
    let x = bits == 32 ? Double(Float(number)) : number
    if x.isNaN { return "NaN" }
    if x.isInfinite { return x < 0 ? "-Inf" : "+Inf" }
    if x == 0 { return x.sign == .minus ? "-0" : "0" }
    let sign = x < 0 ? "-" : ""
    let value = abs(x)
    var text = ""
    for n in 1...(bits == 32 ? 9 : 17) {
      text = String(format: "%.*e", locale: Locale(identifier: "en_US_POSIX"), n - 1, value)
        .lowercased()
      if let parsed = Double(text), (bits == 32 ? Double(Float(parsed)) : parsed) == value { break }
    }
    let parts = text.split(separator: "e")
    let exp = Int(parts[1])!
    let raw = parts[0].filter { $0 != "." }
    var digits = String(raw)
    while digits.last == "0" { digits.removeLast() }
    if exp < -4 || exp >= 6 {
      return sign + String(digits.prefix(1)) + (digits.count > 1 ? "." + digits.dropFirst() : "")
        + "e" + (exp < 0 ? "-" : "+") + String(format: "%02d", abs(exp))
    }
    let point = exp + 1
    if point <= 0 { return sign + "0." + String(repeating: "0", count: -point) + digits }
    if point >= digits.count {
      return sign + digits + String(repeating: "0", count: point - digits.count)
    }
    return sign + digits.prefix(point) + "." + digits.dropFirst(point)
  }
}

func GFormat(_ value: GValue) -> [UInt8] {
  let text: String
  switch value {
  case .nilValue: text = "0x0"
  case .bool(let b): text = b ? "true" : "false"
  case .integer(let i, _, let signed): text = signed ? String(value.intValue) : String(i)
  case .float(let x, let bits): text = GNumeric.floatText(x, bits)
  case .string(let bytes): return bytes
  case .slice(let s): text = "[\(s.length)/\(s.capacity)]0xc000000000"
  case .interface(let b):
    if case .error(let e) = b.value { return e.bytes } else { return GFormat(b.value) }
  case .error(let e): return e.bytes
  default: text = "0xc000000000"
  }
  return Array(text.utf8)
}
func GPrint(_ values: [GValue], _ newline: Bool) throws {
  var bytes: [UInt8] = []
  for i in 0..<values.count {
    if newline && i > 0 { bytes.append(32) }
    bytes += GFormat(values[i])
  }
  if newline { bytes.append(10) }
  FileHandle.standardError.write(Data(bytes))
}
