// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_es384_verify(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.es384_verify", arguments, types)
  }
}
