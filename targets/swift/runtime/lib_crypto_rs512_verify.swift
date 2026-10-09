// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_rs512_verify(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.rs512_verify", arguments, types)
  }
}
