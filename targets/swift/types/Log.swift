// SPDX-License-Identifier: Apache-2.0
import Foundation

/// One std/log/slog record. Levels follow log/slog: -4 debug, 0 info, 4 warn,
/// 8 error. attrs are key/value pairs with group names joined to keys by ".";
/// text is the record rendered like slog's TextHandler without the time.
public struct GoalchemyLogRecord {
  public let level: Int64
  public let unixNano: Int64
  public let message: String
  public let attrs: [(key: String, value: String)]
  public let text: String
  public var time: Date { Date(timeIntervalSince1970: Double(unixNano) / 1e9) }
}

/// Routes records at level and above to handler, synchronously on the source
/// owner's thread. nil restores the default: text written to standard error at
/// level 4 (warn) and above.
public func setLogHandler(_ handler: ((GoalchemyLogRecord) -> Void)?, level: Int64 = 4) {
  GLog.set(handler, level)
}

enum GLog {
  private static let lock = NSLock()
  private static var handler: ((GoalchemyLogRecord) -> Void)?
  private static var level: Int64 = 4
  static func set(_ h: ((GoalchemyLogRecord) -> Void)?, _ l: Int64) {
    lock.lock()
    handler = h
    level = l
    lock.unlock()
  }
  static func enabled(_ l: Int64) -> Bool {
    lock.lock()
    defer { lock.unlock() }
    return l >= level
  }
  static func emit(_ args: [GValue]) {
    lock.lock()
    let h = handler
    lock.unlock()
    let text = args[4].bytes
    guard let h = h else {
      FileHandle.standardError.write(Data(text + [10]))
      return
    }
    var attrs: [(key: String, value: String)] = []
    if case .slice(let s) = args[3], let storage = s.storage {
      var i = 0
      while i + 1 < s.length {
        attrs.append(
          (
            String(decoding: storage.read(s.offset + i).bytes, as: UTF8.self),
            String(decoding: storage.read(s.offset + i + 1).bytes, as: UTF8.self)
          ))
        i += 2
      }
    }
    h(
      GoalchemyLogRecord(
        level: args[0].intValue, unixNano: args[1].intValue,
        message: String(decoding: args[2].bytes, as: UTF8.self), attrs: attrs,
        text: String(decoding: text, as: UTF8.self)))
  }
}
