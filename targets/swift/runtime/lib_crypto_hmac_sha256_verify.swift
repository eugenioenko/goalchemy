// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_hmac_sha256_verify(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.hmac_sha256_verify", arguments, types)
  }
}
