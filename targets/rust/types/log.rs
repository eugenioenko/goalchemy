//! Host log sink for std/log/slog. Levels follow log/slog: -4 debug, 0 info,
//! 4 warn, 8 error. Without a handler, records at warn and above go to stderr.
use super::*;
use std::sync::{Arc, RwLock};
use std::time::{Duration, SystemTime, UNIX_EPOCH};

/// One record; attrs are key/value pairs with group names joined to keys by ".".
#[derive(Clone, Debug)]
pub struct LogRecord {
    pub level: i64,
    pub unix_nano: i64,
    pub message: String,
    pub attrs: Vec<(String, String)>,
    pub text: String,
}

impl LogRecord {
    pub fn time(&self) -> SystemTime {
        if self.unix_nano >= 0 {
            UNIX_EPOCH + Duration::from_nanos(self.unix_nano as u64)
        } else {
            UNIX_EPOCH - Duration::from_nanos(self.unix_nano.unsigned_abs())
        }
    }
}

pub type LogHandler = Arc<dyn Fn(&LogRecord) + Send + Sync>;

static LOG_SINK: RwLock<(Option<LogHandler>, i64)> = RwLock::new((None, 4));

/// Routes records at level and above to handler; None restores stderr.
pub fn set_log_handler(handler: Option<LogHandler>, level: i64) {
    *LOG_SINK.write().unwrap_or_else(|e| e.into_inner()) = (handler, level);
}

pub fn log_enabled(level: i64) -> bool {
    level >= LOG_SINK.read().unwrap_or_else(|e| e.into_inner()).1
}

pub fn log_emit(level: i64, unix_nano: i64, message: &[u8], attrs: &[Rc<[u8]>], text: &[u8]) {
    let handler = LOG_SINK.read().unwrap_or_else(|e| e.into_inner()).0.clone();
    let Some(handler) = handler else {
        let mut line = text.to_vec();
        line.push(b'\n');
        stderr(&line);
        return;
    };
    let s = |b: &[u8]| String::from_utf8_lossy(b).into_owned();
    let record = LogRecord {
        level,
        unix_nano,
        message: s(message),
        attrs: attrs
            .chunks_exact(2)
            .map(|p| (s(&p[0]), s(&p[1])))
            .collect(),
        text: s(text),
    };
    let _ = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| handler(&record)));
}
