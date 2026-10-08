// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_runtime_gosched(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.runtime.gosched", arguments, types)
  }
}
