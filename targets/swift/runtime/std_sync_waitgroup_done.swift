// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_sync_waitgroup_done(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.sync.waitgroup.done", arguments, types)
  }
}
