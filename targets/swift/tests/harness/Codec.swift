// SPDX-License-Identifier: Apache-2.0
import Foundation

struct GHarnessBinding {
  let name: String
  let type: Int
  let kind: String
  let of: String
  init(_ n: String, _ t: Int, _ k: String, _ o: String) {
    name = n
    type = t
    kind = k
    of = o
  }
}
struct GHarnessCase {
  let contract: String
  let bindings: [GHarnessBinding]
  let arguments: [String]
  let outputs: [Int]
  let generic: [String: Int]
  let operation: ([GValue], [Int]) -> GFrame
  init(
    _ c: String, _ b: [GHarnessBinding], _ a: [String], _ o: [Int], _ g: [String: Int],
    _ op: @escaping ([GValue], [Int]) -> GFrame
  ) {
    contract = c
    bindings = b
    arguments = a
    outputs = o
    generic = g
    operation = op
  }
}
enum GCodec {
  static func object(_ raw: Any) -> [String: Any] { raw as? [String: Any] ?? [:] }
  static func integer(_ raw: Any) -> Int64 { Int64(raw as? String ?? String(describing: raw)) ?? 0 }
  static func decode(_ raw: Any, _ type: Int) throws -> GValue {
    let t = GTypes.table[type]
    let obj = object(raw)
    if obj["nil"] as? Bool == true { return GTypes.zero(type) }
    switch t.kind {
    case "int":
      let text = raw as? String ?? String(describing: raw)
      let v = text.hasPrefix("-") ? UInt64(bitPattern: Int64(text)!) : UInt64(text)!
      return GNumeric.normalize(v, t.bits, t.signed)
    case "bool": return .bool((raw as? String) == "true" || (raw as? Bool) == true)
    case "float": return .float(Double(raw as? String ?? "0")!, t.bits)
    case "string":
      if let s = obj["str"] as? String { return .text(s) }
      let hex = Array((obj["hex"] as? String ?? "").utf8)
      var bytes: [UInt8] = []
      for i in stride(from: 0, to: hex.count, by: 2) {
        bytes.append(UInt8(String(decoding: hex[i..<i + 2], as: UTF8.self), radix: 16)!)
      }
      return .string(bytes)
    case "slice":
      let items = obj["slice"] as? [Any] ?? []
      let cap = obj["cap"].map { Int(integer($0)) } ?? items.count
      let b = GBuffer(cap, t.elem)
      for i in 0..<items.count { b.write(i, try decode(items[i], t.elem)) }
      return .slice(GSlice(b, 0, items.count, cap, t.elem))
    case "array":
      let items = obj["array"] as? [Any] ?? []
      let a = GAggregate(type)
      for i in 0..<min(items.count, t.length) { a.buffer!.write(i, try decode(items[i], t.elem)) }
      return .aggregate(a)
    case "map":
      let m = GMap(t.key, t.elem)
      for raw in obj["map"] as? [Any] ?? [] {
        let e = object(raw)
        try GMap.store(.map(m), decode(e["key"]!, t.key), decode(e["value"]!, t.elem))
      }
      return .map(m)
    case "chan":
      let c = GChannel(Int(obj["cap"].map(integer) ?? 0), t.elem)
      c.buffer = try (obj["chan"] as? [Any] ?? []).map { try decode($0, t.elem) }
      c.closed = obj["closed"] as? Bool ?? false
      return .channel(c)
    case "interface":
      if let message = obj["error"] as? String { return GNative.error(message) }
      return .nilValue
    case "pointer": return .pointer(GCell(GTypes.zero(t.elem)))
    case "opaque":
      if t.name == "context.Context" { return try GNative.invoke("std.context.background", [])[0] }
      return GTypes.zero(type)
    default: return GTypes.zero(type)
    }
  }
  static func encode(_ value: GValue, _ type: Int) throws -> Any {
    let t = GTypes.table[type]
    if case .nilValue = value { return ["nil": true] }
    switch t.kind {
    case "int": return t.signed ? String(value.intValue) : String(value.unsigned)
    case "bool": return value.boolValue ? "true" : "false"
    case "float":
      if case .float(let v, let w) = value { return GNumeric.floatText(v, w) }
      return "0"
    case "string": return ["hex": value.bytes.map { String(format: "%02x", $0) }.joined()]
    case "slice":
      guard case .slice(let s) = value else { throw GFault("slice codec") }
      if s.storage == nil { return ["nil": true] }
      return [
        "slice": try GElements(value).map { try encode($0, t.elem) }, "cap": String(s.capacity),
      ]
    case "array": return ["array": try GElements(value).map { try encode($0, t.elem) }]
    case "map":
      guard case .map(let map) = value else { throw GFault("map codec") }
      return [
        "map": try map.entries.filter { $0.live }.map {
          ["key": try encode($0.key, t.key), "value": try encode($0.value, t.elem)]
        }
      ]
    case "chan":
      guard case .channel(let channel) = value else { throw GFault("channel codec") }
      return [
        "chan": try channel.buffer.map { try encode($0, t.elem) }, "cap": String(channel.capacity),
        "closed": channel.closed,
      ]
    case "interface": return ["error": String(decoding: GFormat(value), as: UTF8.self)]
    case "pointer":
      if t.elem >= 0 && GTypes.table[t.elem].name == "crypto.Key" {
        throw GFault("key fixture must be nil")
      }
      return ["zero": true]
    default: return ["zero": true]
    }
  }
  static func execute(_ spec: GHarnessCase, _ args: [GValue]) -> GFrame {
    let c = spec.contract
    let params = spec.generic
    let generic = params.keys.sorted().compactMap { params[$0] }
    func invoke(_ types: [Int] = []) -> GFrame { spec.operation(args, types) }
    switch c {
    case "core.slice.make": return invoke([params["T"]!])
    case "core.slice.to_array":
      let elem = params["T"]!
      let name = "[2]" + GTypes.table[elem].name
      let type =
        GTypes.table.firstIndex { $0.name == name }
        ?? {
          let id = GTypes.table.count
          GTypes.table.append(GType(name, "array", 0, false, elem, -1, 2, [], [], [], [], true))
          return id
        }()
      return invoke([type])
    case "core.string.to_runes": return invoke([GTypes.table.firstIndex { $0.name == "[]rune" }!])
    case "core.map.make": return invoke([params["K"]!, params["V"]!])
    case "core.map.lookup": return invoke([params["V"]!])
    case "core.map.iterate": return invoke([spec.outputs[0]])
    case "core.chan.make": return invoke([params["T"]!])
    case "std.sync.mutex.unlock":
      let f = GFrame(-1)
      f.step = { f, t in
        if f.pc == 0 {
          f.pc = 1
          return .call(GNative.frame("std.sync.mutex.lock", args))
        }
        if f.pc == 1 {
          f.pc = 2
          return .call(spec.operation(args, []))
        }
        return .complete(t.rv)
      }
      return f
    default:
      if c.hasPrefix("core.integer.") || c.hasPrefix("core.float.") { return invoke(generic) }
      return invoke()
    }
  }
}
