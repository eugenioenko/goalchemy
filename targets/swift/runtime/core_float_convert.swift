// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_float_convert(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.float.convert", arguments, types)
  }
}
