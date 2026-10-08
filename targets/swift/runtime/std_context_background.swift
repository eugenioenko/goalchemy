// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_context_background(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.context.background", arguments, types)
  }
}
