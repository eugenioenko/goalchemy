// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_slice_append(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.slice.append", arguments, types)
  }
}
