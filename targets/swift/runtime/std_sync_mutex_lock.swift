// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_sync_mutex_lock(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.sync.mutex.lock", arguments, types)
  }
}
