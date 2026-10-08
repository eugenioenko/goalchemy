// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_aes256_gcm_decrypt(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.aes256_gcm_decrypt", arguments, types)
  }
}
