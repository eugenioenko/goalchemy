// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_float_div(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.float.div", arguments, types)
  }
}
