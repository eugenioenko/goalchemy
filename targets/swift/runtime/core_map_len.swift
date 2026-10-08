// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_map_len(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.map.len", arguments, types)
  }
}
