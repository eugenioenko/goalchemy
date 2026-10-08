// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_random(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.random", arguments, types)
  }
}
