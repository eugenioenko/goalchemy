// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_os_read_file(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.os.read_file", arguments, types)
  }
}
