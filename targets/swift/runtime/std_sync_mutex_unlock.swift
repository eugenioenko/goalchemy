// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_sync_mutex_unlock(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.sync.mutex.unlock", arguments, types)
  }
}
