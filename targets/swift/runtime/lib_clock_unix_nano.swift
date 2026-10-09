// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_clock_unix_nano(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.clock.unix_nano", arguments, types)
  }
}
