// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_map_make(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.map.make", arguments, types)
  }
}
