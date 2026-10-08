// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_errors_new(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.errors.new", arguments, types)
  }
}
