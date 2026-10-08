// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_select(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.select", arguments, types)
  }
}
