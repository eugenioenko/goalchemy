use super::*;
pub fn lib_clock_unix_nano() -> V {
    V::Int(
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_nanos() as i64,
    )
}
