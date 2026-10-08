// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_context_done(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.context.done", arguments, types)
  }
}
