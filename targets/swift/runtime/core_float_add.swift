// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_float_add(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.float.add", arguments, types)
  }
}
