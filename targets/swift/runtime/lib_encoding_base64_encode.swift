// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_encoding_base64_encode(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.encoding.base64_encode", arguments, types)
  }
}
