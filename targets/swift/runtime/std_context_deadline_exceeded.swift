// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_context_deadline_exceeded(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.context.deadline_exceeded", arguments, types)
  }
}
