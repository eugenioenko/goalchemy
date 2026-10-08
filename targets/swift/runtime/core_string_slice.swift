// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_string_slice(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.string.slice", arguments, types)
  }
}
