// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_string_from_rune(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.string.from_rune", arguments, types)
  }
}
