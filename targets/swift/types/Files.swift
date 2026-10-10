// SPDX-License-Identifier: Apache-2.0
import Foundation

enum GFiles {
  static let maxFile = 1 << 30
  static func status(_ e: Int32) -> Int64 {
    switch e {
    case ENOENT: return 1
    case EEXIST: return 2
    case EACCES, EPERM: return 3
    case EISDIR: return 4
    case ENOTDIR: return 5
    case EFBIG: return 6
    case ENOSYS, EOPNOTSUPP: return 7
    case EINVAL: return 8
    default: return 9
    }
  }
  static func path(_ name: GValue) -> ([CChar]?, Int64) {
    let b = name.bytes
    if b.isEmpty { return (nil, 1) }
    if b.contains(0) { return (nil, 8) }
    return (b.map { CChar(bitPattern: $0) } + [0], 0)
  }
  static func readFile(_ name: GValue) -> [GValue] {
    let (p, bad) = path(name)
    guard let p = p else { return [GNative.nilBytes(), .int(bad)] }
    let fd = open(p, O_RDONLY | O_CLOEXEC)
    if fd < 0 { return [GNative.nilBytes(), .int(status(errno))] }
    defer { close(fd) }
    var st = stat()
    if fstat(fd, &st) == 0 && Int(st.st_size) > maxFile { return [GNative.nilBytes(), .int(6)] }
    var data: [UInt8] = []
    var chunk = [UInt8](repeating: 0, count: 1 << 16)
    while true {
      let r = chunk.withUnsafeMutableBytes { read(fd, $0.baseAddress, $0.count) }
      if r < 0 {
        if errno == EINTR { continue }
        return [GNative.nilBytes(), .int(status(errno))]
      }
      if r == 0 { break }
      data.append(contentsOf: chunk[0..<r])
      if data.count > maxFile { return [GNative.nilBytes(), .int(6)] }
    }
    return [GNative.bytes(data), .int(0)]
  }
  static func writeFile(_ name: GValue, _ data: GValue, _ perm: GValue) throws -> [GValue] {
    let (p, bad) = path(name)
    guard let p = p else { return [.int(bad)] }
    let body = try GNative.buffer(data)
    if body.count > maxFile { return [.int(6)] }
    let fd = open(p, O_WRONLY | O_CREAT | O_TRUNC | O_CLOEXEC, mode_t(perm.intValue & 0o777))
    if fd < 0 { return [.int(status(errno))] }
    var code: Int64 = 0
    var done = 0
    while done < body.count {
      let w = body.withUnsafeBytes { write(fd, $0.baseAddress! + done, body.count - done) }
      if w < 0 {
        if errno == EINTR { continue }
        code = status(errno)
        break
      }
      done += w
    }
    if close(fd) != 0 && code == 0 { code = status(errno) }
    return [.int(code)]
  }
}
