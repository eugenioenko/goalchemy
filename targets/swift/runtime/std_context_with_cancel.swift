// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_context_with_cancel(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.context.with_cancel", arguments, types)
  }
}
