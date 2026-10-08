// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func std_errors_unwrap(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("std.errors.unwrap", arguments, types)
  }
}
