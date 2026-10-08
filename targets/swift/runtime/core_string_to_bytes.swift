// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_string_to_bytes(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.string.to_bytes", arguments, types)
  }
}
