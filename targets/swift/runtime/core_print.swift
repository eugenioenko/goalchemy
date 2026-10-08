// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_print(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.print", arguments, types)
  }
}
