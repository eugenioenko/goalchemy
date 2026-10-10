// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_os_write_file(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.os.write_file", arguments, types)
  }
}
