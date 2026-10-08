// SPDX-License-Identifier: Apache-2.0
extension GNative {
  static func core_task_spawn(_ arguments: [GValue], _ types: [Int] = []) -> GFrame {
    frame("core.task.spawn", arguments, types)
  }
}
