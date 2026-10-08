// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_rsa_oaep_decrypt(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.rsa_oaep_decrypt", arguments, types)
  }
}
