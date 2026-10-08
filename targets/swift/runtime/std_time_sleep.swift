// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_time_sleep(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.time.sleep", arguments, types)
  }
}
