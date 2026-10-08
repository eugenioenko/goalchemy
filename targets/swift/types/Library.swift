// SPDX-License-Identifier: Apache-2.0
import Foundation

/// Go strings are arbitrary bytes. String literals use UTF-8; byte construction
/// preserves invalid UTF-8 and NUL without normalization.
public struct GoString: ExpressibleByStringLiteral, Equatable, CustomStringConvertible {
  public var bytes: [UInt8]
  public init(_ string: String) { bytes = Array(string.utf8) }
  public init(bytes: [UInt8]) { self.bytes = bytes }
  public init(stringLiteral value: String) { self.init(value) }
  public var description: String { String(decoding: bytes, as: UTF8.self) }
}
public struct GoalchemyFailure: Error, CustomStringConvertible {
  public let kind: String
  public let message: String
  public let fields: [String: Any]
  public init(_ kind: String, _ message: String, _ fields: [String: Any] = [:]) {
    self.kind = kind
    self.message = message
    self.fields = fields
  }
  public var description: String { message }
}
public final class CancellationToken {
  private let lock = NSLock()
  private var canceled = false
  private var hook: (() -> Void)?
  public init() {}
  public var isCanceled: Bool {
    lock.lock()
    defer { lock.unlock() }
    return canceled
  }
  public func cancel() {
    lock.lock()
    canceled = true
    let h = hook
    lock.unlock()
    h?()
  }
  func install(_ fn: (() -> Void)?) {
    lock.lock()
    hook = fn
    let already = canceled
    lock.unlock()
    if already { fn?() }
  }
}
public struct CallbackRequest {
  public let bytes: [UInt8]
  public let cancellation: CancellationToken
}
public typealias CallbackProvider = (CallbackRequest, @escaping (Result<[UInt8], Error>) -> Void) ->
  (() -> Void)?
public struct CallOptions {
  public var cancellation: CancellationToken
  public var timeoutNanoseconds: Int64?
  public var callbacks: [String: CallbackProvider]
  public init(
    cancellation: CancellationToken = CancellationToken(), timeoutNanoseconds: Int64? = nil,
    callbacks: [String: CallbackProvider] = [:]
  ) {
    self.cancellation = cancellation
    self.timeoutNanoseconds = timeoutNanoseconds
    self.callbacks = callbacks
  }
}
public final class Operation<T> {
  public let cancellation: CancellationToken
  private let condition = NSCondition()
  private var body: (() throws -> T)?
  private var running = false
  private var result: Result<T, Error>?
  init(_ cancellation: CancellationToken, _ body: @escaping () throws -> T) {
    self.cancellation = cancellation
    self.body = body
  }
  public func cancel() { cancellation.cancel() }
  public func wait() throws -> T {
    if Thread.current.threadDictionary["goalchemy.provider"] != nil {
      throw GoalchemyFailure("host", "reentrant source call from callback provider")
    }
    condition.lock()
    while running && result == nil { condition.wait() }
    if let result = result {
      condition.unlock()
      return try result.get()
    }
    running = true
    let call = body!
    body = nil
    condition.unlock()
    let result = Result { try call() }
    condition.lock()
    self.result = result
    running = false
    condition.broadcast()
    condition.unlock()
    return try result.get()
  }
  public func value() async throws -> T {
    try await withCheckedThrowingContinuation { continuation in
      DispatchQueue.global().async { continuation.resume(with: Result { try self.wait() }) }
    }
  }
}
enum GLibrary {
  static func provider<T>(_ body: () -> T) -> T {
    let thread = Thread.current
    let prior = thread.threadDictionary["goalchemy.provider"]
    thread.threadDictionary["goalchemy.provider"] = true
    defer {
      if let prior = prior {
        thread.threadDictionary["goalchemy.provider"] = prior
      } else {
        thread.threadDictionary.removeObject(forKey: "goalchemy.provider")
      }
    }
    return body()
  }
  static let lock = NSRecursiveLock()
  static var retained: [UUID: [GValue]] = [:]
  static func prepare(_ convert: () throws -> [GValue]) throws -> [GValue] {
    if Thread.current.threadDictionary["goalchemy.provider"] != nil {
      throw GoalchemyFailure("host", "reentrant source call from callback provider")
    }
    lock.lock()
    defer { lock.unlock() }
    return try convert()
  }
  static func submit<T>(
    _ args: [GValue], _ options: CallOptions, _ reset: @escaping () -> Void,
    _ initialize: @escaping () -> GFrame, _ start: @escaping ([GValue], GValue) -> GFrame,
    _ decode: @escaping ([GValue]) throws -> T
  ) -> Operation<T> {
    lock.lock()
    let id = UUID()
    retained[id] = args
    lock.unlock()
    let held = GLibraryRetention(id)
    return Operation(options.cancellation) {
      _ = held
      lock.lock()
      defer {
        retained[id] = nil
        GHeap.collect(retained.values.flatMap { $0 })
        lock.unlock()
      }
      reset()
      let owner = GOwner(host: true)
      owner.callbacks = options.callbacks
      let elem =
        GTypes.table.firstIndex { $0.kind == "struct" && $0.fields.isEmpty }
        ?? GNative.addPrimitive("struct {}", "struct", 0, false)
      let context = GContext(elem)
      context.owner = owner
      options.cancellation.install { owner.mailbox.publish { context.cancel() } }
      defer { options.cancellation.install(nil) }
      if options.cancellation.isCanceled { context.cancel() }
      if let delay = options.timeoutNanoseconds {
        if delay <= 0 {
          context.cancel(true)
        } else {
          context.timer = owner.timer(delay, retained: [.opaque(context)]) { context.cancel(true) }
        }
      }
      let frame = GFrame(-1)
      frame.roots = { args + [.opaque(context)] }
      frame.step = { frame, task in
        if frame.pc == 0 {
          frame.pc = 1
          return .call(initialize())
        }
        if frame.pc == 1 {
          frame.pc = 2
          return .call(start(args, .opaque(context)))
        }
        return .complete(task.rv)
      }
      do { return try decode(owner.run(frame)) } catch let p as GPanic {
        throw GoalchemyFailure("panic", GPanicText(p))
      } catch let e as GFault { throw GoalchemyFailure("host", e.message) } catch let e as GFatal {
        throw GoalchemyFailure("runtime", e.message)
      }
    }
  }
}
final class GLibraryRetention {
  let id: UUID
  init(_ id: UUID) { self.id = id }
  deinit {
    GLibrary.lock.lock()
    GLibrary.retained[id] = nil
    GLibrary.lock.unlock()
  }
}
