// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_generate_p521(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.generate_p521", arguments, types)
  }
}
