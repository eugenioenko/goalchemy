// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_es256_sign(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.es256_sign", arguments, types)
  }
}
