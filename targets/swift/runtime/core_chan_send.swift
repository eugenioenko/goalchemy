// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_chan_send(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.chan.send", arguments, types)
  }
}
