//! Serves runtime conformance requests for the Rust target over JSON Lines
//! on standard input and output.
#![allow(warnings)]

mod codec;
mod harness_gen;
mod json;
mod rt;

use json::J;
use rt::*;
use std::io::{BufRead, Write};

pub struct H {
    pub lets: J,
    pub after: std::cell::RefCell<Vec<(String, J)>>,
}

impl H {
    pub fn let_(&self, n: &str) -> &J {
        self.lets.get(n).unwrap_or(&J::Null)
    }

    pub fn after(&self, n: &str, v: J) {
        self.after.borrow_mut().push((n.to_string(), v));
    }
}

fn serve(req: &J) -> J {
    let mut resp = vec![("v".to_string(), J::Num("1".into())), ("id".to_string(), req.get("id").cloned().unwrap_or(J::Null))];
    let fail = |mut resp: Vec<(String, J)>, msg: String| {
        resp.push(("status".into(), J::Str("harness_failure".into())));
        resp.push(("error".into(), J::Str(msg)));
        J::Obj(resp)
    };
    if req.get("v").map(|v| v.str()) != Some("1") {
        return fail(resp, "unsupported protocol version".into());
    }
    let name = req.get("case").map(|c| c.str()).unwrap_or("");
    let Some((_, f)) = harness_gen::CASES.iter().find(|(n, _)| *n == name) else {
        return fail(resp, format!("unknown case {}", name));
    };
    let h = H { lets: req.get("let").cloned().unwrap_or(J::Obj(Vec::new())), after: Default::default() };
    reset_scheduler();
    let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| f(&h)));
    match r {
        Ok(results) => {
            resp.push(("status".into(), J::Str("returned".into())));
            resp.push(("results".into(), J::Arr(results)));
            resp.push(("after".into(), J::Obj(h.after.into_inner())));
        }
        Err(e) => {
            if let Some(p) = e.downcast_ref::<GoPanicPayload>() {
                let msg = format_panic_value(&panic_value(&V::Obj(p.0)));
                resp.push(("status".into(), J::Str("panic".into())));
                resp.push(("panic".into(), J::Str(String::from_utf8_lossy(&msg).into_owned())));
            } else if e.is::<BlockedPayload>() {
                resp.push(("status".into(), J::Str("blocked".into())));
            } else {
                let msg = e.downcast_ref::<String>().cloned().or_else(|| e.downcast_ref::<&str>().map(|s| s.to_string()));
                return fail(resp, msg.unwrap_or_else(|| "host panic".into()));
            }
        }
    }
    J::Obj(resp)
}

fn main() {
    std::panic::set_hook(Box::new(|_| {}));
    let t = std::thread::Builder::new().stack_size(1 << 29).spawn(|| {
        let stdin = std::io::stdin();
        let mut out = std::io::stdout().lock();
        for line in stdin.lock().lines() {
            let Ok(line) = line else { break };
            let line = line.trim();
            if line.is_empty() {
                continue;
            }
            let resp = match json::parse(line) {
                Ok(req) => serve(&req),
                Err(e) => J::obj(vec![("v", J::Num("1".into())), ("status", J::Str("harness_failure".into())), ("error", J::Str(e))]),
            };
            let mut s = String::new();
            json::write(&resp, &mut s);
            s.push('\n');
            let _ = out.write_all(s.as_bytes());
            let _ = out.flush();
        }
    });
    let _ = t.unwrap().join();
}
