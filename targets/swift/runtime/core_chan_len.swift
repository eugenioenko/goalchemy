// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_chan_len(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.chan.len", arguments, types)
  }
}
