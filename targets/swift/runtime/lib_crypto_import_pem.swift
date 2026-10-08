// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_crypto_import_pem(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.crypto.import_pem", arguments, types)
  }
}
