// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_hkdf_sha256(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.hkdf_sha256", arguments, types)
  }
}
