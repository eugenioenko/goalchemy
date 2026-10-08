// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_string_compare(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.string.compare", arguments, types)
  }
}
