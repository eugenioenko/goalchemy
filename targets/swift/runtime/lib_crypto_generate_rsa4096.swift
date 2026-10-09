// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_generate_rsa4096(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.generate_rsa4096", arguments, types)
  }
}
