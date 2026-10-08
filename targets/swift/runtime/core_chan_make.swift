// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_chan_make(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.chan.make", arguments, types)
  }
}
