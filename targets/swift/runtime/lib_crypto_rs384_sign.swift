// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_rs384_sign(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.rs384_sign", arguments, types)
  }
}
