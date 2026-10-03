//! Providers execute outside source locks, and return only after resource release.
#![cfg(feature = "native")]
use super::*;
fn callback_zero(e: V) -> Vec<V> {
    vec![BYTE_NIL, e]
}
pub fn lib_callback_request(t: &Rc<Task>, ctx: V, name: V, payload: V) {
    if ctx.is_nil() {
        t.set_rv(callback_zero(std_errors_new(s(b"callback: invalid input"))));
        return;
    }
    check_context(&ctx);
    let e = std_context_context_err(ctx.clone());
    if !e.is_nil() {
        t.set_rv(callback_zero(e));
        return;
    }
    let name = name.bytes();
    let provider = std::str::from_utf8(&name)
        .ok()
        .filter(|s| !s.is_empty() && s.len() <= 256)
        .and_then(|name| LIBRARY_PROVIDERS.with(|p| p.borrow().get(name).cloned()));
    let body = match native_bytes(&payload) {
        Ok(b) => b,
        Err(_) => {
            t.set_rv(callback_zero(std_errors_new(s(
                b"callback: invalid payload",
            ))));
            return;
        }
    };
    let provider = match provider {
        Some(p) => p,
        None => {
            t.set_rv(callback_zero(std_errors_new(s(
                b"callback: provider not found",
            ))));
            return;
        }
    };
    let cancel = Cancellation::default();
    let stopped = cancel.clone();
    let token = register_host(
        t,
        host_boundary(ctx.clone(), None),
        vec![ctx],
        callback_zero,
        |w| match w.as_slice() {
            [HostWire::Bool(true), HostWire::Bytes(b)] => {
                vec![native_slice_bytes(b.clone()), V::Nil]
            }
            [HostWire::Bool(false), HostWire::Bytes(b)] => callback_zero(std_errors_new(s(b))),
            _ => host_fault("provider wire decode"),
        },
        move || stopped.cancel(),
        || {},
    );
    launch_host(token, move || {
        let result = provider(ProviderRequest {
            payload: body,
            cancellation: cancel,
        });
        drop(provider);
        match result {
            Ok(b) if b.len() <= 128 * 1024 => vec![HostWire::Bool(true), HostWire::Bytes(b)],
            Ok(_) => vec![
                HostWire::Bool(false),
                HostWire::Bytes(b"provider: response limit".to_vec()),
            ],
            Err(ProviderError::Rejected(s)) => {
                vec![HostWire::Bool(false), HostWire::Bytes(s.into_bytes())]
            }
            Err(ProviderError::Fault(s)) => host_fault(&s),
        }
    });
}
