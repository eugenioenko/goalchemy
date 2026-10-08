// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_integer_andnot(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.integer.andnot", arguments, types)
  }
}
