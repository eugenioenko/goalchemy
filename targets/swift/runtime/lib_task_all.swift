// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func lib_task_all(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("lib.task.all", arguments, types)
  }
}
