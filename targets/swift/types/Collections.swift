// SPDX-License-Identifier: Apache-2.0
import Foundation

#if os(Linux)
  import Glibc
#else
  import Darwin
#endif

extension GSlice {
  static func make(_ length: GValue, _ capacity: GValue, _ elem: Int) throws -> GValue {
    let n = try GBounds.size(length, "makeslice: len out of range")
    let c = try GBounds.size(capacity, "makeslice: cap out of range")
    if n > c { throw GPanic.runtime("makeslice: cap out of range") }
    return .slice(GSlice(GBuffer(c, elem), 0, n, c, elem))
  }
  static func cell(_ value: GValue, _ index: GValue) throws -> GCell {
    guard case .slice(let s) = value else { throw GFault("slice representation") }
    let n = try GBounds.index(index, s.length)
    return s.storage!.cell(s.offset + n)
  }
  // A retained native view snapshots the source before mutation. Swift COW also
  // handles distinct GBuffer wrappers sharing an Array allocation; Go aliases
  // still observe mutation through their shared GBuffer object.
  static func byteSource(_ value: GValue) -> ArraySlice<UInt8>? {
    switch value {
    case .slice(let s):
      if let bytes = s.storage?.bytes { return bytes[s.offset..<s.offset + s.length] }
      if s.length == 0 { return [] }
    case .string(let bytes): return bytes[...]
    case .aggregate(let a):
      if let bytes = a.buffer?.bytes { return bytes[...] }
    default: break
    }
    return nil
  }
  private static func isByte(_ elem: Int) -> Bool {
    let t = GTypes.table[elem]
    return t.kind == "int" && t.bits == 8 && !t.signed
  }
  // Do not retain an ArraySlice for a move inside the same Go backing object:
  // that would force Swift COW to duplicate the entire native byte allocation.
  private static func moveWithin(_ buffer: GBuffer, from: Int, to: Int, count: Int) {
    if count == 0 || from == to { return }
    buffer.bytes!.withUnsafeMutableBufferPointer { bytes in
      _ = memmove(bytes.baseAddress!.advanced(by: to), bytes.baseAddress!.advanced(by: from), count)
    }
  }
  static func append(_ value: GValue, _ source: GValue) throws -> GValue {
    if case .slice(let s) = value, isByte(s.elem) {
      if case .slice(let src) = source, let buffer = s.storage,
        src.storage === buffer, buffer.bytes != nil
      {
        if src.length == 0 { return value }
        let (n, overflow) = s.length.addingReportingOverflow(src.length)
        if overflow { throw GFault("slice allocation exceeds limits") }
        if n <= s.capacity {
          moveWithin(buffer, from: src.offset, to: s.offset + s.length, count: src.length)
          return .slice(GSlice(buffer, s.offset, n, s.capacity, s.elem))
        }
      }
      if let bytes = byteSource(source) { return try appendBytes(s, bytes) }
    }
    // Keep source validation and recursive value copying in the generic path.
    return try append(value, GElements(source))
  }
  private static func appendBytes(_ s: GSlice, _ values: ArraySlice<UInt8>) throws -> GValue {
    if values.isEmpty { return .slice(s) }
    let (n, overflow) = s.length.addingReportingOverflow(values.count)
    if overflow { throw GFault("slice allocation exceeds limits") }
    var storage = s.storage
    var offset = s.offset
    var capacity = s.capacity
    if n > capacity {
      let doubled = capacity.multipliedReportingOverflow(by: 2)
      capacity = max(n, max(1, doubled.overflow ? Int.max : doubled.partialValue))
      let new = GBuffer(capacity, s.elem)
      if s.length > 0 {
        let bytes = s.storage!.bytes!
        new.bytes!.replaceSubrange(0..<s.length, with: bytes[s.offset..<s.offset + s.length])
      }
      storage = new
      offset = 0
    }
    storage!.bytes!.replaceSubrange(offset + s.length..<offset + n, with: values)
    return .slice(GSlice(storage, offset, n, capacity, s.elem))
  }
  static func append(_ value: GValue, _ values: [GValue]) throws -> GValue {
    guard case .slice(let s) = value else { throw GFault("slice representation") }
    if isByte(s.elem) {
      return try appendBytes(s, values.map { UInt8(truncatingIfNeeded: $0.unsigned) }[...])
    }
    if values.isEmpty { return value }
    let (n, overflow) = s.length.addingReportingOverflow(values.count)
    if overflow { throw GFault("slice allocation exceeds limits") }
    var storage = s.storage
    var offset = s.offset
    var capacity = s.capacity
    if n > capacity {
      let doubled = capacity.multipliedReportingOverflow(by: 2)
      capacity = max(n, max(1, doubled.overflow ? Int.max : doubled.partialValue))
      let new = GBuffer(capacity, s.elem)
      for i in 0..<s.length { new.write(i, GCopy(s.storage!.read(s.offset + i))) }
      storage = new
      offset = 0
    }
    // Snapshot first: appending overlapping storage must not corrupt the source.
    let copied = values.map(GCopy)
    for i in 0..<copied.count { storage!.write(offset + s.length + i, copied[i]) }
    return .slice(GSlice(storage, offset, n, capacity, s.elem))
  }
  static func copy(_ dst: GValue, _ src: GValue) throws -> GValue {
    guard case .slice(let d) = dst else { throw GFault("slice representation") }
    if isByte(d.elem) {
      if case .slice(let s) = src, let buffer = d.storage,
        s.storage === buffer, buffer.bytes != nil
      {
        let n = min(d.length, s.length)
        moveWithin(buffer, from: s.offset, to: d.offset, count: n)
        return .int(Int64(n))
      }
      if let bytes = byteSource(src) {
        let n = min(d.length, bytes.count)
        if n > 0 { d.storage!.bytes!.replaceSubrange(d.offset..<d.offset + n, with: bytes.prefix(n)) }
        return .int(Int64(n))
      }
    }
    let elements = try GElements(src)
    let n = min(d.length, elements.count)
    let snapshot = elements.prefix(n).map(GCopy)
    for i in 0..<n { d.storage!.write(d.offset + i, snapshot[i]) }
    return .int(Int64(n))
  }
  static func reslice(_ source: GValue, _ low: GValue?, _ high: GValue?, _ maximum: GValue?) throws
    -> GValue
  {
    var v = source
    if case .pointer(let p) = v { v = p.value }
    if case .string(let bytes) = v {
      let lo = low?.intValue ?? 0
      let hi = high?.intValue ?? Int64(bytes.count)
      try GBounds.slice(lo, hi, nil, bytes.count, "length")
      return .string(Array(bytes[Int(lo)..<Int(hi)]))
    }
    let s: GSlice
    switch v {
    case .slice(let value): s = value
    case .aggregate(let a):
      guard let buffer = a.buffer else { throw GFault("array representation") }
      s = GSlice(buffer, 0, buffer.count, buffer.count, buffer.elem)
    default: throw GPanic.runtime("invalid memory address or nil pointer dereference")
    }
    let lo = low?.intValue ?? 0
    let hi = high?.intValue ?? Int64(s.length)
    let mx = maximum?.intValue ?? Int64(s.capacity)
    try GBounds.slice(lo, hi, maximum == nil ? nil : mx, s.capacity, "capacity")
    return .slice(GSlice(s.storage, s.offset + Int(lo), Int(hi - lo), Int(mx - lo), s.elem))
  }
}

final class GMapEntry {
  let key: GValue
  var value: GValue
  var live = true
  init(_ key: GValue, _ value: GValue) {
    self.key = key
    self.value = value
  }
}
final class GMap: GManaged {
  let key: Int
  let elem: Int
  var entries: [GMapEntry] = []
  init(_ key: Int, _ elem: Int) {
    self.key = key
    self.elem = elem
    GHeap.track(self)
  }
  func find(_ key: GValue) throws -> GMapEntry? {
    try GHashable(key)
    return try entries.first {
      if !$0.live { return false }
      return try GEqual($0.key, key)
    }
  }
  static func lookup(_ value: GValue, _ key: GValue, _ elem: Int) throws -> (GValue, Bool) {
    try GHashable(key)
    guard case .map(let map) = value, let entry = try map.find(key) else {
      return (GTypes.zero(elem), false)
    }
    return (GCopy(entry.value), true)
  }
  static func store(_ value: GValue, _ key: GValue, _ element: GValue) throws {
    guard case .map(let map) = value else { throw GPanic.plain("assignment to entry in nil map") }
    if let entry = try map.find(key) {
      entry.value = GCopy(element)
    } else {
      map.entries.append(GMapEntry(GCopy(key), GCopy(element)))
    }
  }
  static func delete(_ value: GValue, _ key: GValue) throws {
    try GHashable(key)
    if case .map(let map) = value, let entry = try map.find(key) {
      entry.live = false
      entry.value = .nilValue
    }
  }
  func children() -> [GValue] { entries.filter { $0.live }.flatMap { [$0.key, $0.value] } }
  func releaseEdges() { entries = [] }
}
func GHashable(_ value: GValue) throws {
  switch value {
  case .slice: throw GPanic.runtime("hash of unhashable type slice")
  case .map: throw GPanic.runtime("hash of unhashable type map")
  case .function: throw GPanic.runtime("hash of unhashable type func")
  case .interface(let box):
    if box.type >= 0 && !GTypes.table[box.type].comparable {
      throw GPanic.runtime("hash of unhashable type " + GTypes.table[box.type].name)
    }
    try GHashable(box.value)
  case .aggregate(let a):
    for v in a.children() { if case .pointer(let cell) = v { try GHashable(cell.value) } }
  default: break
  }
}
final class GMapIterator: GManaged {
  var entries: [GMapEntry]
  var index = 0
  init(_ value: GValue) {
    if case .map(let map) = value { entries = map.entries.filter { $0.live } } else { entries = [] }
    GHeap.track(self)
  }
  static func next(_ value: GValue) throws -> (Bool, GValue, GValue) {
    guard case .iterator(let iter) = value else { throw GFault("iterator representation") }
    while iter.index < iter.entries.count {
      let entry = iter.entries[iter.index]
      iter.index += 1
      if entry.live { return (true, GCopy(entry.key), GCopy(entry.value)) }
    }
    return (false, .nilValue, .nilValue)
  }
  func children() -> [GValue] { entries.flatMap { [$0.key, $0.value] } }
  func releaseEdges() { entries = [] }
}

func GLength(_ value: GValue, _ capacity: Bool) throws -> GValue {
  switch value {
  case .string(let bytes): return .int(Int64(bytes.count))
  case .slice(let s): return .int(Int64(capacity ? s.capacity : s.length))
  case .aggregate(let a): return .int(Int64(a.buffer?.count ?? 0))
  case .pointer(let cell): return try GLength(cell.value, capacity)
  case .map(let map): return .int(Int64(map.entries.filter { $0.live }.count))
  case .channel(let channel): return .int(Int64(capacity ? channel.capacity : channel.buffer.count))
  case .nilValue: return .int(0)
  default: throw GFault("len/cap representation")
  }
}
func GClear(_ value: GValue) throws {
  switch value {
  case .map(let map):
    for entry in map.entries {
      entry.live = false
      entry.value = .nilValue
    }
    map.entries = []
  case .slice(let s): for i in 0..<s.length { s.storage!.write(s.offset + i, GTypes.zero(s.elem)) }
  case .nilValue: break
  default: throw GFault("clear representation")
  }
}

enum GString {
  static func index(_ s: GValue, _ i: GValue) throws -> GValue {
    let bytes = s.bytes
    return .integer(UInt64(bytes[try GBounds.index(i, bytes.count)]), 8, false)
  }
  static func decode(_ value: GValue, _ position: GValue) throws -> (GValue, GValue) {
    let bytes = value.bytes
    let i = try GBounds.index(position, value.bytes.count)
    let first = bytes[i]
    if first < 128 { return (.int(Int64(first)), .int(1)) }
    let width: Int
    if first >= 0xc2 && first <= 0xdf {
      width = 2
    } else if first >= 0xe0 && first <= 0xef {
      width = 3
    } else if first >= 0xf0 && first <= 0xf4 {
      width = 4
    } else {
      return (.int(65533), .int(1))
    }
    if i + width > bytes.count { return (.int(65533), .int(1)) }
    var rune = UInt32(first & (UInt8(0x7f) >> width))
    for j in 1..<width {
      let b = bytes[i + j]
      if b < 0x80 || b > 0xbf { return (.int(65533), .int(1)) }
      rune = (rune << 6) | UInt32(b & 0x3f)
    }
    if (width == 2 && rune < 128) || (width == 3 && rune < 2048) || (width == 4 && rune < 65536)
      || rune > 0x10ffff || (rune >= 0xd800 && rune <= 0xdfff)
    {
      return (.int(65533), .int(1))
    }
    return (.int(Int64(rune)), .int(Int64(width)))
  }
  static func rune(_ value: GValue) -> [UInt8] {
    let r = value.intValue
    let scalar =
      (r < 0 || r > 0x10ffff || (r >= 0xd800 && r <= 0xdfff))
      ? Unicode.Scalar(65533)! : Unicode.Scalar(UInt32(r))!
    return Array(String(scalar).utf8)
  }
}

func GConvert(_ value: GValue, _ type: Int, _ conversion: Int) throws -> GValue {
  let target = GTypes.table[type]
  switch conversion {
  case 1, 9: return value
  case 2:
    return GNumeric.normalize(
      GNumeric.signedInteger(value) ? UInt64(bitPattern: value.intValue) : value.unsigned,
      target.bits, target.signed)
  case 3: return .string(GString.rune(value))
  case 4:
    let b = GBuffer(bytes: value.bytes, elem: target.elem)
    return .slice(GSlice(b, 0, b.count, b.count, target.elem))
  case 5: return .string(try GElements(value).map { UInt8(truncatingIfNeeded: $0.unsigned) })
  case 6:
    var runes: [GValue] = []
    var position = 0
    while position < value.bytes.count {
      let (r, w) = try GString.decode(value, .int(Int64(position)))
      runes.append(r)
      position += Int(w.intValue)
    }
    let buffer = GBuffer(runes.count, target.elem)
    for i in 0..<runes.count { buffer.write(i, GNumeric.normalize(runes[i].unsigned, 32, true)) }
    return .slice(GSlice(buffer, 0, runes.count, runes.count, target.elem))
  case 7: return .string(try GElements(value).flatMap { GString.rune($0) })
  case 8:
    let elements = try GElements(value)
    if elements.count < target.length {
      throw GPanic.runtime(
        "cannot convert slice with length \(elements.count) to array or pointer to array with length \(target.length)",
        "runtime.boundsError"
      )
    }
    let aggregate = GAggregate(type)
    for i in 0..<target.length { aggregate.buffer!.write(i, GCopy(elements[i])) }
    return .aggregate(aggregate)
  case 10: return GNumeric.convert(value, target.bits, target.signed, target.kind == "float")
  default: throw GFault("numeric conversion")
  }
}
