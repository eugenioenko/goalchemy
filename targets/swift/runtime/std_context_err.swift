// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_context_err(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.context.err", arguments, types)
  }
}
