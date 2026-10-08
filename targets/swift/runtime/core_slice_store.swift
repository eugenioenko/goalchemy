// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_slice_store(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.slice.store", arguments, types)
  }
}
