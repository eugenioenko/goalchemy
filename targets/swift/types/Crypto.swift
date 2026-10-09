// SPDX-License-Identifier: Apache-2.0
import Foundation
import GoalchemyNative

struct GReject: Error {
  let message: String
  init(_ message: String = "crypto: invalid input or key") { self.message = message }
}
final class GCryptoKey: GOpaque {
  private let condition = NSCondition()
  private var pointer: UnsafeMutableRawPointer?
  private var leases = 0
  private var closed = false
  var escaped = false
  init(_ pointer: UnsafeMutableRawPointer) {
    self.pointer = pointer
    super.init()
  }
  func snapshot() throws -> UnsafeMutableRawPointer {
    condition.lock()
    defer { condition.unlock() }
    if closed { throw GReject("crypto: key is closed") }
    guard let p = pointer, let retained = gcn_key_retain(p) else { throw GReject() }
    leases += 1
    return retained
  }
  func release(_ pointer: UnsafeMutableRawPointer) {
    gcn_key_release(pointer)
    condition.lock()
    leases -= 1
    condition.broadcast()
    condition.unlock()
  }
  func close() {
    condition.lock()
    closed = true
    while leases > 0 { condition.wait() }
    let pointer = pointer
    self.pointer = nil
    condition.unlock()
    if let pointer = pointer { gcn_key_release(pointer) }
  }
  func beginClose() {
    condition.lock()
    closed = true
    condition.unlock()
  }
  deinit { if let pointer = pointer { gcn_key_release(pointer) } }
}
final class GWeakCryptoKey {
  weak var value: GCryptoKey?
  init(_ value: GCryptoKey) { self.value = value }
}
public final class GoalchemyKey {
  let key: GCryptoKey
  let cell: GCell
  private let root: UUID
  init(_ key: GCryptoKey, _ cell: GCell) {
    self.key = key
    self.cell = cell
    key.escaped = true
    root = GHeap.retainExternal(.pointer(cell))
  }
  deinit { GHeap.releaseExternal(root) }
  public func close() { key.close() }
}
struct GCryptoInput {
  let operation: String
  var bytes: [[UInt8]] = []
  var numbers: [Int64] = []
  var keys: [(GCryptoKey, UnsafeMutableRawPointer)] = []
  func release() { for (key, pointer) in keys { key.release(pointer) } }
}
enum GCryptoOutput {
  case bytes([UInt8])
  case text(String)
  case strings([String])
  case key(UnsafeMutableRawPointer)
  case boolean(Bool)
  case empty
  case error(String)
}
enum GCrypto {
  static func crc32(_ value: GValue) throws -> GValue {
    let bytes = try GNative.buffer(value)
    let result = bytes.withUnsafeBufferPointer { gcn_crc32($0.baseAddress, $0.count) }
    return .integer(UInt64(result), 32, false)
  }
  static func key(_ value: GValue) throws -> GCryptoKey {
    guard case .pointer(let cell) = value, case .opaque(let opaque) = cell.value,
      let key = opaque as? GCryptoKey
    else { throw GReject() }
    return key
  }
  static func prepare(_ contract: String, _ args: [GValue]) throws -> GCryptoInput {
    let op = String(contract.dropFirst("lib.crypto.".count))
    var input = GCryptoInput(operation: op)
    do {
      for value in args {
        switch value {
        case .slice:
          let bytes = try GNative.buffer(value)
          if bytes.count > 64 << 20
            && !(op == "aes256_gcm_decrypt" && bytes.count <= (64 << 20) + 16)
          {
            throw GReject()
          }
          input.bytes.append(bytes)
        case .string(let bytes): input.bytes.append(bytes)
        case .integer: input.numbers.append(value.intValue)
        default:
          let key = try self.key(value)
          input.keys.append((key, try key.snapshot()))
        }
      }
      return input
    } catch {
      input.release()
      throw error
    }
  }
  static func zero(_ op: String) -> GValue {
    if op.contains("verify") { return .bool(false) }
    if op.hasPrefix("generate_") || op == "import_pem" { return .nilValue }
    if op == "public_pem" || op == "private_pem" { return .text("") }
    if op == "public_jwk" {
      let elem =
        GTypes.table.firstIndex { $0.kind == "string" }
        ?? GNative.addPrimitive("string", "string", 0, false)
      return .slice(GSlice(nil, 0, 0, 0, elem))
    }
    return GNative.nilBytes()
  }
  static func decode(_ op: String, _ output: GCryptoOutput, _ owner: GOwner? = nil) -> [GValue] {
    switch output {
    case .bytes(let bytes): return [GNative.bytes(bytes), .nilValue]
    case .text(let text): return [.text(text), .nilValue]
    case .strings(let values): return [GNative.strings(values), .nilValue]
    case .key(let pointer):
      let key = GCryptoKey(pointer)
      owner?.keys.append(GWeakCryptoKey(key))
      return [.pointer(GCell(.opaque(key))), .nilValue]
    case .boolean(let value): return [.bool(value), .nilValue]
    case .empty: return []
    case .error(let message): return [zero(op), GNative.error(message)]
    }
  }
  static func start(_ contract: String, _ args: [GValue], _ task: GTask) throws -> GAction {
    let op = String(contract.dropFirst("lib.crypto.".count))
    if op == "close" {
      if case .nilValue = args[0] { return .resume([]) }
      let key = try self.key(args[0])
      key.beginClose()
      let operation = task.owner.newOperation(task)
      DispatchQueue.global().async {
        key.close()
        task.owner.mailbox.publish { task.owner.finish(operation, []) }
      }
      return .park
    }
    let input: GCryptoInput
    do { input = try prepare(contract, args) } catch let reject as GReject {
      return .resume([zero(op), GNative.error(reject.message)])
    }
    let operation = task.owner.newOperation(task)
    operation.retained = args
    DispatchQueue.global().async {
      let output = work(input)
      input.release()
      task.owner.mailbox.publish {
        if !task.owner.active {
          if case .key(let pointer) = output { gcn_key_release(pointer) }
          task.owner.finish(operation, [])
        } else {
          task.owner.finish(operation, decode(op, output, task.owner))
        }
      }
    }
    return .park
  }
  static func invoke(_ contract: String, _ args: [GValue]) throws -> [GValue] {
    let op = String(contract.dropFirst("lib.crypto.".count))
    if op == "close" {
      if case .nilValue = args[0] { return [] }
      try key(args[0]).close()
      return []
    }
    do {
      let input = try prepare(contract, args)
      defer { input.release() }
      return decode(op, work(input), GOwner.current)
    } catch let reject as GReject { return [zero(op), GNative.error(reject.message)] }
  }
  static func work(_ input: GCryptoInput) -> GCryptoOutput {
    do { return try operation(input) } catch let e as GReject { return .error(e.message) } catch {
      return .error("crypto: native operation failed")
    }
  }
  static func operation(_ input: GCryptoInput) throws -> GCryptoOutput {
    let op = input.operation
    let b = input.bytes
    func require(_ test: Bool) throws { if !test { throw GReject() } }
    func kind(_ index: Int, _ expected: Int32, _ privateKey: Bool = false) throws
      -> UnsafeMutableRawPointer
    {
      let k = input.keys[index].1
      try require(gcn_key_kind(k) == expected && (!privateKey || gcn_key_private(k) == 1))
      return k
    }
    func digest(_ data: [UInt8]) -> [UInt8] {
      var out = [UInt8](repeating: 0, count: 32)
      let ok = data.withUnsafeBufferPointer { p in
        out.withUnsafeMutableBufferPointer { gcn_sha256(p.baseAddress, p.count, $0.baseAddress) }
      }
      return ok == 1 ? out : []
    }
    func hmac(_ key: [UInt8], _ data: [UInt8]) -> [UInt8] {
      var out = [UInt8](repeating: 0, count: 32)
      let ok = key.withUnsafeBufferPointer { k in
        data.withUnsafeBufferPointer { d in
          out.withUnsafeMutableBufferPointer {
            gcn_hmac(k.baseAddress, k.count, d.baseAddress, d.count, $0.baseAddress)
          }
        }
      }
      return ok == 1 ? out : []
    }
    switch op {
    case "random":
      let n = input.numbers[0]
      try require(n >= 0 && n <= 64 << 20)
      var out = [UInt8](repeating: 0, count: Int(n))
      let ok = out.withUnsafeMutableBufferPointer { gcn_random($0.baseAddress, $0.count) }
      try require(ok == 1)
      return .bytes(out)
    case "sha256": return .bytes(digest(b[0]))
    case "hmac_sha256": return .bytes(hmac(b[0], b[1]))
    case "hmac_sha256_verify":
      try require(b[2].count == 32)
      let expected = hmac(b[0], b[1])
      var diff: UInt8 = 0
      for i in 0..<32 { diff |= expected[i] ^ b[2][i] }
      return .boolean(diff == 0)
    case "hkdf_sha256":
      let count = input.numbers[0]
      try require(count >= 0 && count <= 8160)
      let prk = hmac(b[1], b[0])
      var prev: [UInt8] = []
      var out: [UInt8] = []
      var index: UInt8 = 1
      while out.count < count {
        prev = hmac(prk, prev + b[2] + [index])
        out += prev.prefix(min(32, Int(count) - out.count))
        index &+= 1
      }
      return .bytes(out)
    case "aes256_gcm_encrypt", "aes256_gcm_decrypt":
      let decrypt = op.hasSuffix("decrypt")
      try require(
        b[0].count == 32 && b[1].count == 12 && b[3].count <= 64 << 20
          && (!decrypt || b[2].count >= 16))
      var out = [UInt8](repeating: 0, count: b[2].count + 16)
      var count = out.count
      let ok = b[0].withUnsafeBufferPointer { k in
        b[1].withUnsafeBufferPointer { iv in
          b[2].withUnsafeBufferPointer { d in
            b[3].withUnsafeBufferPointer { aad in
              out.withUnsafeMutableBufferPointer { o in
                gcn_aes(
                  decrypt ? 1 : 0, k.baseAddress, iv.baseAddress, d.baseAddress, d.count,
                  aad.baseAddress, aad.count, o.baseAddress, &count)
              }
            }
          }
        }
      }
      if ok != 1 {
        throw GReject(decrypt ? "crypto: authentication failed" : "crypto: native operation failed")
      }
      return .bytes(Array(out.prefix(count)))
    case "generate_rsa2048", "generate_rsa4096", "generate_p256", "generate_p384", "generate_p521":
      guard let kind = Int32(op.dropFirst(op.hasPrefix("generate_rsa") ? 12 : 10)),
        let key = gcn_key_generate(kind)
      else {
        throw GReject("crypto: key generation failed")
      }
      return .key(key)
    case "import_pem":
      try require(b[0].count <= 65536)
      let text = String(decoding: b[0], as: UTF8.self).trimmingCharacters(
        in: .whitespacesAndNewlines)
      let types = ["PUBLIC KEY", "PRIVATE KEY", "RSA PUBLIC KEY", "RSA PRIVATE KEY", "CERTIFICATE"]
      guard
        let type = types.first(where: {
          text.hasPrefix("-----BEGIN " + $0 + "-----\n")
            || text.hasPrefix("-----BEGIN " + $0 + "-----\r\n")
        })
      else { throw GReject() }
      try require(
        text.components(separatedBy: "-----BEGIN ").count == 2
          && text.hasSuffix("-----END " + type + "-----") && !text.contains(":"))
      let bytes = Array(text.utf8)
      guard let key = bytes.withUnsafeBufferPointer({ gcn_key_import($0.baseAddress, $0.count) })
      else { throw GReject() }
      return .key(key)
    case "public_pem", "private_pem":
      let key = input.keys[0].1
      if op == "private_pem" { try require(gcn_key_private(key) == 1) }
      var out: UnsafeMutablePointer<UInt8>?
      var count = 0
      try require(gcn_key_export(key, op == "private_pem" ? 1 : 0, &out, &count) == 1)
      defer { gcn_free(out) }
      return .text(String(decoding: UnsafeBufferPointer(start: out, count: count), as: UTF8.self))
    case "public_jwk":
      let key = input.keys[0].1
      let k = gcn_key_kind(key)
      let width = gcn_key_width(key)
      func component(_ index: Int32) throws -> String {
        var bytes = [UInt8](repeating: 0, count: 512)
        var count = bytes.count
        let ok = bytes.withUnsafeMutableBufferPointer {
          gcn_key_component(key, index, $0.baseAddress, &count)
        }
        try require(ok == 1)
        return Data(bytes.prefix(count)).base64EncodedString().replacingOccurrences(
          of: "+", with: "-"
        ).replacingOccurrences(of: "/", with: "_").replacingOccurrences(of: "=", with: "")
      }
      return .strings(
        k == 1
          ? ["RSA", "", try component(0), try component(1), "", ""]
          : [
            "EC", width == 32 ? "P-256" : width == 48 ? "P-384" : "P-521", "", "", try component(2),
            try component(3),
          ])
    case "rsa_oaep_encrypt", "rsa_oaep_decrypt":
      let decrypt = op.hasSuffix("decrypt")
      let key = try kind(0, 1, decrypt)
      let width = Int(gcn_key_width(key))
      try require(decrypt ? b[0].count == width : b[0].count <= width - 42)
      var out = [UInt8](repeating: 0, count: width)
      var count = out.count
      let ok = b[0].withUnsafeBufferPointer { d in
        out.withUnsafeMutableBufferPointer { o in
          gcn_oaep(decrypt ? 1 : 0, key, d.baseAddress, d.count, o.baseAddress, &count)
        }
      }
      if ok != 1 {
        throw GReject(decrypt ? "crypto: decryption failed" : "crypto: native operation failed")
      }
      return .bytes(Array(out.prefix(count)))
    case "rs256_sign", "rs384_sign", "rs512_sign", "es256_sign", "es384_sign", "es512_sign",
      "rs256_verify", "rs384_verify", "rs512_verify", "es256_verify", "es384_verify",
      "es512_verify":
      let rsa = op.hasPrefix("rs")
      let bits = Int32(op.dropFirst(2).prefix(3))!
      let verify = op.hasSuffix("verify")
      let key = try kind(0, rsa ? 1 : 2, !verify)
      let width = Int(gcn_key_width(key))
      if !rsa { try require(width == (bits == 256 ? 32 : bits == 384 ? 48 : 66)) }
      if verify {
        try require(b[1].count == (rsa ? width : 2 * width))
        let ok = b[0].withUnsafeBufferPointer { d in
          b[1].withUnsafeBufferPointer { s in
            gcn_verify(rsa ? 1 : 2, bits, key, d.baseAddress, d.count, s.baseAddress, s.count)
          }
        }
        return .boolean(ok == 1)
      }
      var out = [UInt8](repeating: 0, count: 512)
      var count = out.count
      let ok = b[0].withUnsafeBufferPointer { d in
        out.withUnsafeMutableBufferPointer { o in
          gcn_sign(rsa ? 1 : 2, bits, key, d.baseAddress, d.count, o.baseAddress, &count)
        }
      }
      try require(ok == 1)
      return .bytes(Array(out.prefix(count)))
    case "ecdh":
      let privateKey = try kind(0, 2, true)
      let publicKey = try kind(1, 2)
      let width = Int(gcn_key_width(privateKey))
      try require(width == Int(gcn_key_width(publicKey)))
      var out = [UInt8](repeating: 0, count: 66)
      var count = out.count
      let ok = out.withUnsafeMutableBufferPointer {
        gcn_ecdh(privateKey, publicKey, $0.baseAddress, &count)
      }
      try require(ok == 1 && count == width)
      return .bytes(Array(out.prefix(count)))
    default: throw GReject("crypto: unknown operation")
    }
  }
}
extension GOwner {
  func newOperation(_ task: GTask) -> GHostOperation {
    operationID += 1
    let op = GHostOperation(operationID, task)
    operations[op.id] = op
    return op
  }
  func finish(_ operation: GHostOperation, _ values: [GValue]) {
    guard operations.removeValue(forKey: operation.id) === operation else { return }
    if active { ready(operation.task, values) }
  }
}
