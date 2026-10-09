// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_es512_sign(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.es512_sign", arguments, types)
  }
}
