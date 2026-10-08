// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_integer_shr(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.integer.shr", arguments, types)
  }
}
