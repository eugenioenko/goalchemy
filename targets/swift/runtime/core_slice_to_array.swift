// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_slice_to_array(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.slice.to_array", arguments, types)
  }
}
