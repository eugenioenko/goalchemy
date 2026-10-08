// SPDX-License-Identifier: Apache-2.0
import Foundation

final class GChannelWait {
  let task: GTask
  let send: Bool
  let value: GValue
  let selection: GSelection?
  let index: Int
  var active = true
  init(
    _ task: GTask, _ send: Bool, _ value: GValue, _ selection: GSelection? = nil, _ index: Int = 0
  ) {
    self.task = task
    self.send = send
    self.value = value
    self.selection = selection
    self.index = index
  }
  func complete(_ value: GValue, _ ok: Bool) {
    if !active { return }
    active = false
    if let group = selection {
      group.commit(index, value, ok)
    } else {
      task.cleanup = nil
      task.owner.ready(task, send ? [] : [value, .bool(ok)])
    }
  }
}
final class GSelection {
  let task: GTask
  var waits: [GChannelWait] = []
  var committed = false
  init(_ task: GTask) { self.task = task }
  func commit(_ index: Int, _ value: GValue, _ ok: Bool) {
    if committed { return }
    committed = true
    for wait in waits { wait.active = false }
    task.cleanup = nil
    task.owner.ready(task, [.int(Int64(index)), value, .bool(ok)])
  }
  func clear() {
    for wait in waits { wait.active = false }
    waits = []
  }
}
struct GSelectCase {
  let channel: GValue
  let send: Bool
  let value: GValue
  init(_ c: GValue, _ send: Bool, _ value: GValue) {
    channel = c
    self.send = send
    self.value = value
  }
}
final class GChannel: GManaged {
  let capacity: Int
  let elem: Int
  var buffer: [GValue] = []
  var sends: [GChannelWait] = []
  var receives: [GChannelWait] = []
  var closed = false
  init(_ capacity: Int, _ elem: Int) {
    self.capacity = capacity
    self.elem = elem
    GHeap.track(self)
  }
  static func make(_ capacity: GValue, _ elem: Int) throws -> GValue {
    if capacity.intValue < 0 { throw GPanic.plain("makechan: size out of range") }
    return .channel(GChannel(try GBounds.size(capacity, "makechan: size out of range"), elem))
  }
  func prune() {
    sends.removeAll { !$0.active }
    receives.removeAll { !$0.active }
  }
  func ready(_ send: Bool) -> Bool {
    prune()
    if send { return closed || !receives.isEmpty || buffer.count < capacity }
    return closed || !buffer.isEmpty || !sends.isEmpty
  }
  func commitSend(_ value: GValue) throws {
    prune()
    if closed { throw GPanic.plain("send on closed channel") }
    if !receives.isEmpty {
      receives.removeFirst().complete(GCopy(value), true)
    } else {
      buffer.append(GCopy(value))
    }
  }
  func commitReceive() -> (GValue, Bool) {
    prune()
    if !buffer.isEmpty {
      let value = buffer.removeFirst()
      if !sends.isEmpty && !closed {
        let wait = sends.removeFirst()
        buffer.append(GCopy(wait.value))
        wait.complete(.nilValue, true)
      }
      return (value, true)
    }
    if !sends.isEmpty && !closed {
      let wait = sends.removeFirst()
      wait.complete(.nilValue, true)
      return (GCopy(wait.value), true)
    }
    return (GTypes.zero(elem), false)
  }
  static func send(_ value: GValue, _ element: GValue, _ task: GTask) throws -> GAction {
    guard case .channel(let channel) = value else { return .park }
    if channel.ready(true) {
      try channel.commitSend(element)
      return .resume([])
    }
    let wait = GChannelWait(task, true, GCopy(element))
    channel.sends.append(wait)
    task.cleanup = { wait.active = false }
    return .park
  }
  static func receive(_ value: GValue, _ task: GTask) throws -> GAction {
    guard case .channel(let channel) = value else { return .park }
    if channel.ready(false) {
      let (v, ok) = channel.commitReceive()
      return .resume([v, .bool(ok)])
    }
    let wait = GChannelWait(task, false, .nilValue)
    channel.receives.append(wait)
    task.cleanup = { wait.active = false }
    return .park
  }
  static func close(_ value: GValue, _ owner: GOwner) throws {
    guard case .channel(let channel) = value else { throw GPanic.plain("close of nil channel") }
    if channel.closed { throw GPanic.plain("close of closed channel") }
    channel.closed = true
    channel.prune()
    for wait in channel.receives where wait.active {
      let buffered = !channel.buffer.isEmpty
      let v = buffered ? channel.buffer.removeFirst() : GTypes.zero(channel.elem)
      wait.complete(v, buffered)
    }
    for wait in channel.sends where wait.active {
      wait.active = false
      wait.task.resumePanic = GPanic.plain("send on closed channel")
      wait.task.cleanup = nil
      if let group = wait.selection {
        group.commit(wait.index, .nilValue, false)
      } else {
        owner.ready(wait.task)
      }
    }
    channel.prune()
  }
  static func select(_ cases: [GSelectCase], _ hasDefault: Bool, _ task: GTask) throws -> GAction {
    var ready: [Int] = []
    for i in 0..<cases.count {
      if case .channel(let c) = cases[i].channel, c.ready(cases[i].send) { ready.append(i) }
    }
    if !ready.isEmpty {
      let index = ready[task.owner.choose(ready.count)]
      let c = cases[index]
      guard case .channel(let channel) = c.channel else { throw GFault("select representation") }
      if c.send {
        try channel.commitSend(c.value)
        return .resume([.int(Int64(index)), .nilValue, .bool(false)])
      }
      let (value, ok) = channel.commitReceive()
      return .resume([.int(Int64(index)), value, .bool(ok)])
    }
    if hasDefault { return .resume([.int(-1), .nilValue, .bool(false)]) }
    let group = GSelection(task)
    for i in 0..<cases.count {
      let c = cases[i]
      if case .channel(let channel) = c.channel {
        let wait = GChannelWait(task, c.send, GCopy(c.value), group, i)
        group.waits.append(wait)
        if c.send { channel.sends.append(wait) } else { channel.receives.append(wait) }
      }
    }
    task.cleanup = { group.clear() }
    return .park
  }
  func children() -> [GValue] { buffer + sends.filter { $0.active }.map { $0.value } }
  func releaseEdges() {
    buffer = []
    sends = []
    receives = []
  }
}

final class GMutex: GOpaque {
  var locked = false
  var waiters: [GTask] = []
  override func valueCopy() -> GOpaque? {
    let m = GMutex()
    m.locked = locked
    return m
  }
  func lock(_ task: GTask) -> GAction {
    if !locked {
      locked = true
      return .resume([])
    }
    waiters.append(task)
    return .park
  }
  func unlock() throws {
    if !locked { throw GFatal("sync: unlock of unlocked mutex") }
    if waiters.isEmpty {
      locked = false
    } else {
      let t = waiters.removeFirst()
      t.owner.ready(t, [])
    }
  }
  override func releaseEdges() { waiters = [] }
}
final class GWaitGroup: GOpaque {
  var count: Int64 = 0
  var waiters: [GTask] = []
  override func valueCopy() -> GOpaque? {
    let w = GWaitGroup()
    w.count = count
    return w
  }
  func add(_ n: Int64) throws {
    count &+= n
    if count < 0 { throw GPanic.string("sync: negative WaitGroup counter") }
    if count == 0 {
      let tasks = waiters
      waiters = []
      for task in tasks { task.owner.ready(task, []) }
    }
  }
  func wait(_ task: GTask) -> GAction {
    if count == 0 { return .resume([]) }
    waiters.append(task)
    return .park
  }
  override func releaseEdges() { waiters = [] }
}
final class GContext: GOpaque {
  let done: GChannel
  var error: GValue = .nilValue
  var parent: GContext?
  var descendants: [GContext] = []
  var timer: GTimer?
  var hooks: [UUID: () -> Void] = [:]
  weak var owner: GOwner?
  init(_ elem: Int) {
    done = GChannel(0, elem)
    super.init()
  }
  func cancel(_ deadline: Bool = false) {
    if case .nilValue = error {
      error = deadline ? GNative.deadlineError : GNative.canceledError
      if let owner = owner {
        try? GChannel.close(.channel(done), owner)
      } else {
        done.closed = true
      }
      timer?.active = false
      timer = nil
      for hook in hooks.values { hook() }
      for child in descendants { child.cancel(deadline) }
      descendants = []
      if let p = parent {
        p.descendants.removeAll { $0 === self }
        parent = nil
      }
    }
  }
  override func children() -> [GValue] {
    [.channel(done), error] + descendants.map { .opaque($0) } + (parent.map { [.opaque($0)] } ?? [])
  }
  override func releaseEdges() {
    parent = nil
    descendants = []
    timer = nil
    hooks = [:]
    done.releaseEdges()
  }
}
