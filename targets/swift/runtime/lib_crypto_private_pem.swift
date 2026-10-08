// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_private_pem(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.private_pem", arguments, types)
  }
}
