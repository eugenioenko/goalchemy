use super::*;
pub fn lib_log_enabled(level: V) -> V {
    V::Bool(log_enabled(level.i()))
}
