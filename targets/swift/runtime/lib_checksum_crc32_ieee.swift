// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_checksum_crc32_ieee(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.checksum.crc32_ieee", arguments, types)
  }
}
