// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_map_store(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.map.store", arguments, types)
  }
}
