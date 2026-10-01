//! Byte-exact output.
use std::io::Write;

pub fn stderr(b: &[u8]) {
    let mut e = std::io::stderr().lock();
    let _ = e.write_all(b);
    let _ = e.flush();
}

pub fn stdout(b: &[u8]) {
    let mut o = std::io::stdout().lock();
    let _ = o.write_all(b);
    let _ = o.flush();
}
