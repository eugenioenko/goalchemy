// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_log_emit(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.log.emit", arguments, types)
  }
}
