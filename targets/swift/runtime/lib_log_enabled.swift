// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_log_enabled(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.log.enabled", arguments, types)
  }
}
