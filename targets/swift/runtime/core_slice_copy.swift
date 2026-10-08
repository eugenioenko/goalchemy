// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_slice_copy(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.slice.copy", arguments, types)
  }
}
