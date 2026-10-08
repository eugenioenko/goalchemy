// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_sync_waitgroup_add(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.sync.waitgroup.add", arguments, types)
  }
}
