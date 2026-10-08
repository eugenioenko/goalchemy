// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_close(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.close", arguments, types)
  }
}
