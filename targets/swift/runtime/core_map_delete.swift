// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_map_delete(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.map.delete", arguments, types)
  }
}
