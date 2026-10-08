// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_callback_request(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.callback.request", arguments, types)
  }
}
