// SPDX-License-Identifier: Apache-2.0
import Foundation

enum GValue {
  case nilValue
  case bool(Bool)
  case integer(UInt64, Int, Bool)
  case float(Double, Int)
  case string([UInt8])
  case aggregate(GAggregate)
  case slice(GSlice)
  case map(GMap)
  case pointer(GCell)
  case interface(GInterface)
  case function(GFunction)
  case channel(GChannel)
  case iterator(GMapIterator)
  case opaque(GOpaque)
  case error(GError)

  var boolValue: Bool {
    if case .bool(let v) = self { return v }
    return false
  }
  var unsigned: UInt64 {
    if case .integer(let v, _, _) = self { return v }
    return 0
  }
  var intValue: Int64 {
    if case .integer(let v, let w, let signed) = self {
      if signed && w < 64 && v & (UInt64(1) << (w - 1)) != 0 {
        return Int64(bitPattern: v | (~UInt64(0) << w))
      }
      return Int64(bitPattern: v)
    }
    return 0
  }
  var bytes: [UInt8] {
    if case .string(let v) = self { return v }
    return []
  }
  static func int(_ v: Int64) -> GValue { return .integer(UInt64(bitPattern: v), 64, true) }
  static func text(_ v: String) -> GValue { return .string(Array(v.utf8)) }
}

protocol GManaged: AnyObject {
  func children() -> [GValue]
  func releaseEdges()
}
final class GWeak {
  weak var value: GManaged?
  init(_ value: GManaged) { self.value = value }
}
enum GHeap {
  static var allocations: [GWeak] = []
  static var roots: [GCell] = []
  // Public native wrappers preserve source pointer identity. Their cells stay
  // traced even between calls; registration/removal may occur on host threads.
  private static let externalLock = NSLock()
  private static var external: [UUID: GValue] = [:]
  static func retainExternal(_ value: GValue) -> UUID {
    externalLock.lock()
    defer { externalLock.unlock() }
    let id = UUID()
    external[id] = value
    return id
  }
  static func releaseExternal(_ id: UUID) {
    externalLock.lock()
    external[id] = nil
    externalLock.unlock()
  }
  private static func externalRoots() -> [GValue] {
    externalLock.lock()
    defer { externalLock.unlock() }
    return Array(external.values)
  }
  static func track(_ v: GManaged) { allocations.append(GWeak(v)) }
  static func collect(_ additional: [GValue] = []) {
    var seen = Set<ObjectIdentifier>()
    var todo =
      roots.map { GValue.pointer($0) } + additional + externalRoots()
      + GTypes.table.flatMap { $0.methods.values.map { GValue.function($0) } } + [
        GNative.canceledError, GNative.deadlineError,
      ] + (GNative.background.map { [GValue.opaque($0)] } ?? [])
    while let v = todo.popLast() {
      let object: GManaged?
      switch v {
      case .aggregate(let o): object = o
      case .slice(let s): object = s.storage
      case .map(let o): object = o
      case .pointer(let o): object = o
      case .interface(let o): object = o
      case .function(let o): object = o
      case .channel(let o): object = o
      case .iterator(let o): object = o
      case .opaque(let o): object = o
      default: object = nil
      }
      if let o = object, seen.insert(ObjectIdentifier(o)).inserted { todo += o.children() }
    }
    for weak in allocations {
      if let o = weak.value, !seen.contains(ObjectIdentifier(o)) { o.releaseEdges() }
    }
    allocations.removeAll { $0.value == nil }
  }
  static var live: Int { allocations.filter { $0.value != nil }.count }
}

final class GCell: GManaged {
  private var stored: GValue
  private var get: (() -> GValue)?
  private var set: ((GValue) -> Void)?
  var backing: [GValue] = []
  var value: GValue {
    get { get?() ?? stored }
    set { if let set = set { set(newValue) } else { stored = newValue } }
  }
  init(_ v: GValue) {
    stored = v
    GHeap.track(self)
  }
  init(_ get: @escaping () -> GValue, _ set: @escaping (GValue) -> Void) {
    stored = .nilValue
    self.get = get
    self.set = set
    GHeap.track(self)
  }
  func children() -> [GValue] { [value] + backing }
  func releaseEdges() {
    stored = .nilValue
    get = nil
    set = nil
    backing = []
  }
}

final class GType {
  let name: String
  let kind: String
  let bits: Int
  let signed: Bool
  let elem: Int
  let key: Int
  let length: Int
  let fields: [Int]
  let fieldNames: [String]
  let blank: [Bool]
  let requiredMethods: [String]
  let comparable: Bool
  let implementedInterfaces: Set<Int>
  let missingInterfaceMethods: [Int: String]
  let nativeMethods: Set<String>
  var methods: [String: GFunction] = [:]
  init(
    _ name: String, _ kind: String, _ bits: Int, _ signed: Bool, _ elem: Int,
    _ key: Int, _ length: Int, _ fields: [Int], _ names: [String], _ blank: [Bool],
    _ required: [String], _ comparable: Bool, _ implemented: [Int] = [],
    _ missing: [Int: String] = [:], _ native: [String] = []
  ) {
    self.name = name
    self.kind = kind
    self.bits = bits
    self.signed = signed
    self.elem = elem
    self.key = key
    self.length = length
    self.fields = fields
    fieldNames = names
    self.blank = blank
    requiredMethods = required
    self.comparable = comparable
    implementedInterfaces = Set(implemented)
    missingInterfaceMethods = missing
    nativeMethods = Set(native)
  }
}
enum GTypes {
  static var table: [GType] = []
  static func zero(_ id: Int) -> GValue {
    let t = table[id]
    switch t.kind {
    case "bool": return .bool(false)
    case "int": return .integer(0, t.bits, t.signed)
    case "float": return .float(0, t.bits)
    case "string": return .string([])
    case "struct", "array": return .aggregate(GAggregate(id))
    case "slice": return .slice(GSlice(nil, 0, 0, 0, t.elem))
    case "opaque":
      if t.name == "sync.Mutex" { return .opaque(GMutex()) }
      if t.name == "sync.WaitGroup" { return .opaque(GWaitGroup()) }
      return .nilValue
    default: return .nilValue
    }
  }
}

final class GBuffer: GManaged {
  var bytes: [UInt8]?
  var cells: [GCell]
  let elem: Int
  init(_ n: Int, _ elem: Int) {
    self.elem = elem
    let t = GTypes.table[elem]
    if t.kind == "int" && t.bits == 8 && !t.signed {
      bytes = [UInt8](repeating: 0, count: n)
      cells = []
    } else {
      bytes = nil
      cells = (0..<n).map { _ in GCell(GTypes.zero(elem)) }
    }
    GHeap.track(self)
  }
  init(bytes: [UInt8], elem: Int) {
    self.bytes = bytes
    self.elem = elem
    cells = []
    GHeap.track(self)
  }
  var count: Int { bytes?.count ?? cells.count }
  func read(_ index: Int) -> GValue {
    if let bytes = bytes { return .integer(UInt64(bytes[index]), 8, false) }
    return cells[index].value
  }
  func write(_ index: Int, _ v: GValue) {
    if bytes != nil {
      bytes![index] = UInt8(truncatingIfNeeded: v.unsigned)
    } else {
      GStore(cells[index], v)
    }
  }
  func cell(_ index: Int) -> GCell {
    if bytes != nil {
      let cell = GCell({ self.read(index) }, { self.write(index, $0) })
      cell.backing = [.slice(GSlice(self, 0, count, count, elem))]
      return cell
    }
    return cells[index]
  }
  func children() -> [GValue] { cells.map { .pointer($0) } }
  func releaseEdges() {
    cells.removeAll()
    bytes = nil
  }
}

func GGlobalCell(_ value: GValue) -> GCell {
  let cell = GCell(value)
  GHeap.roots.append(cell)
  return cell
}

final class GAggregate: GManaged {
  let type: Int
  var fields: [GCell]
  var buffer: GBuffer?
  init(_ type: Int) {
    self.type = type
    let t = GTypes.table[type]
    if t.kind == "array" {
      fields = []
      buffer = GBuffer(t.length, t.elem)
    } else {
      fields = t.fields.map { GCell(GTypes.zero($0)) }
      buffer = nil
    }
    GHeap.track(self)
  }
  static func field(_ v: GValue, _ n: Int) throws -> GCell {
    guard case .aggregate(let o) = v else {
      throw GPanic.runtime("invalid memory address or nil pointer dereference")
    }
    return o.fields[n]
  }
  static func element(_ v: GValue, _ index: GValue) throws -> GCell {
    guard case .aggregate(let o) = v, let b = o.buffer else {
      throw GPanic.runtime("invalid memory address or nil pointer dereference")
    }
    return b.cell(try GBounds.index(index, b.count))
  }
  func children() -> [GValue] {
    fields.map { .pointer($0) }
      + (buffer.map { [.slice(GSlice($0, 0, $0.count, $0.count, $0.elem))] } ?? [])
  }
  func releaseEdges() {
    fields = []
    buffer = nil
  }
}

struct GSlice {
  let storage: GBuffer?
  let offset: Int
  let length: Int
  let capacity: Int
  let elem: Int
  init(_ storage: GBuffer?, _ offset: Int, _ length: Int, _ capacity: Int, _ elem: Int) {
    self.storage = storage
    self.offset = offset
    self.length = length
    self.capacity = capacity
    self.elem = elem
  }
}

func GCopy(_ v: GValue) -> GValue {
  if case .aggregate(let o) = v {
    let result = GAggregate(o.type)
    if let src = o.buffer, let dst = result.buffer {
      for i in 0..<src.count { dst.write(i, GCopy(src.read(i))) }
    } else {
      for i in 0..<o.fields.count { GStore(result.fields[i], o.fields[i].value) }
    }
    return .aggregate(result)
  }
  if case .opaque(let o) = v, let copy = o.valueCopy() { return .opaque(copy) }
  return v
}
func GStore(_ dst: GCell, _ src: GValue) {
  if case .aggregate(let a) = dst.value, case .aggregate(let b) = src {
    if a === b { return }
    if let d = a.buffer, let s = b.buffer {
      for i in 0..<d.count { d.write(i, GCopy(s.read(i))) }
    } else {
      for i in 0..<a.fields.count { GStore(a.fields[i], b.fields[i].value) }
    }
  } else {
    dst.value = GCopy(src)
  }
}
enum GPointer {
  static func cell(_ v: GValue) throws -> GCell {
    guard case .pointer(let c) = v else {
      throw GPanic.runtime("invalid memory address or nil pointer dereference")
    }
    return c
  }
}

class GOpaque: GManaged {
  init() { GHeap.track(self) }
  func valueCopy() -> GOpaque? { nil }
  func children() -> [GValue] { [] }
  func releaseEdges() {}
}
final class GError {
  let message: String
  let bytes: [UInt8]
  let type: String
  let identity: Bool
  init(_ message: String, _ type: String = "*errors.errorString", _ identity: Bool = true) {
    self.message = message
    self.bytes = Array(message.utf8)
    self.type = type
    self.identity = identity
  }
  init(bytes: [UInt8]) {
    self.bytes = bytes
    self.message = String(decoding: bytes, as: UTF8.self)
    self.type = "*errors.errorString"
    self.identity = true
  }
}

func GEqual(_ a: GValue, _ b: GValue) throws -> Bool {
  switch (a, b) {
  case (.nilValue, .nilValue): return true
  case (.nilValue, .slice(let s)), (.slice(let s), .nilValue): return s.storage == nil
  case (.slice(let x), .slice(let y)) where x.storage == nil || y.storage == nil:
    return x.storage == nil && y.storage == nil
  case (.bool(let x), .bool(let y)): return x == y
  case (.integer(let x, _, _), .integer(let y, _, _)): return x == y
  case (.float(let x, _), .float(let y, _)): return x == y
  case (.string(let x), .string(let y)): return x == y
  case (.pointer(let x), .pointer(let y)): return x === y
  case (.channel(let x), .channel(let y)): return x === y
  case (.opaque(let x), .opaque(let y)): return x === y
  case (.aggregate(let x), .aggregate(let y)):
    let t = GTypes.table[x.type]
    if let xb = x.buffer, let yb = y.buffer {
      for i in 0..<xb.count { if try !GEqual(xb.read(i), yb.read(i)) { return false } }
    } else {
      for i in 0..<x.fields.count where !t.blank[i] {
        if try !GEqual(x.fields[i].value, y.fields[i].value) { return false }
      }
    }
    return true
  case (.interface(let x), .interface(let y)):
    if x.type != y.type { return false }
    if x.type >= 0 && !GTypes.table[x.type].comparable {
      throw GPanic.runtime("comparing uncomparable type " + GTypes.table[x.type].name)
    }
    return try GEqual(x.value, y.value)
  case (.error(let x), .error(let y)):
    guard x.type == y.type else { return false }
    return x.identity ? x === y : x.message == y.message
  case (.slice, _), (.map, _), (.function, _):
    if case .nilValue = b {
      if case .slice(let s) = a { return s.storage == nil }
      return false
    }
    throw GPanic.runtime("comparing uncomparable type")
  default: return false
  }
}

func GElements(_ v: GValue) throws -> [GValue] {
  switch v {
  case .string(let bytes): return bytes.map { .integer(UInt64($0), 8, false) }
  case .slice(let s): return (0..<s.length).map { s.storage!.read(s.offset + $0) }
  case .aggregate(let a): return (0..<(a.buffer?.count ?? 0)).map { a.buffer!.read($0) }
  default: throw GFault("expected slice, array or string")
  }
}

enum GBounds {
  static func index(_ v: GValue, _ n: Int) throws -> Int {
    let i = v.intValue
    if !GNumeric.signedInteger(v) && v.unsigned > UInt64(Int64.max) {
      throw GPanic.runtime(
        "index out of range [\(v.unsigned)] with length \(n)", "runtime.boundsError")
    }
    if i < 0 || UInt64(i) >= UInt64(n) {
      throw GPanic.runtime(
        i < 0 ? "index out of range [\(i)]" : "index out of range [\(i)] with length \(n)",
        "runtime.boundsError")
    }
    return Int(i)
  }
  static func size(_ v: GValue, _ message: String) throws -> Int {
    let i = v.intValue
    if i < 0 { throw GPanic.runtime(message) }
    if i > Int64(Int.max) { throw GFault("allocation exceeds host limits") }
    return Int(i)
  }
}

extension GBounds {
  static func slice(_ lo: Int64, _ hi: Int64, _ maximum: Int64?, _ limit: Int, _ word: String)
    throws
  {
    func bad(_ text: String) throws {
      throw GPanic.runtime("slice bounds out of range " + text, "runtime.boundsError")
    }
    if let mx = maximum {
      if mx < 0 { try bad("[::\(mx)]") }
      if mx > limit { try bad("[::\(mx)] with \(word) \(limit)") }
      if hi < 0 { try bad("[:\(hi):]") }
      if hi > mx { try bad("[:\(hi):\(mx)]") }
      if lo < 0 { try bad("[\(lo)::]") }
      if lo > hi { try bad("[\(lo):\(hi):]") }
    } else {
      if hi < 0 { try bad("[:\(hi)]") }
      if hi > limit { try bad("[:\(hi)] with \(word) \(limit)") }
      if lo < 0 { try bad("[\(lo):]") }
      if lo > hi { try bad("[\(lo):\(hi)]") }
    }
  }
}
