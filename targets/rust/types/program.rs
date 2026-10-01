//! Program entry, globals, and unrecovered-panic reports.
use super::*;

thread_local! {
    static GLOBALS: Cell<*mut FrData> = Cell::new(std::ptr::null_mut());
}

/// Allocates the program's global variables as a permanent root.
pub fn init_globals(n: usize) {
    GLOBALS.with(|g| g.set(Fr::new(n).leak()));
}

#[inline]
pub fn gg(i: usize) -> V {
    GLOBALS.with(|g| unsafe { (&(*g.get()).v)[i].clone() })
}

#[inline]
pub fn gs(i: usize, x: V) {
    let old = GLOBALS.with(|g| unsafe { std::mem::replace(&mut (&mut (*g.get()).v)[i], x) });
    drop(old);
}

fn indented(b: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(b.len());
    for &c in b {
        out.push(c);
        if c == b'\n' {
            out.push(b'\t');
        }
    }
    out
}

pub fn format_panic_value(v: &V) -> Vec<u8> {
    let Some((t, x)) = unbox(v) else { return b"nil".to_vec() };
    let _root = temp_root(&[v.clone()]);
    for m in ["Error", "String"] {
        if let Some((_, code)) = t.method(m) {
            return indented(&code(&[], vec![x]).bytes());
        }
    }
    let builtin = !t.name.contains('.');
    let wrap = |inner: Vec<u8>, quote: bool| -> Vec<u8> {
        if builtin {
            return inner;
        }
        let mut b = t.name.as_bytes().to_vec();
        b.extend_from_slice(if quote { b"(\"" } else { b"(" });
        b.extend(inner);
        b.extend_from_slice(if quote { b"\")" } else { b")" });
        b
    };
    match t.basic {
        "string" => wrap(indented(&x.bytes()), true),
        "bool" => wrap(if x.b() { b"true".to_vec() } else { b"false".to_vec() }, false),
        "int" => wrap(x.i().to_string().into_bytes(), false),
        "uint" => wrap((x.i() as u64).to_string().into_bytes(), false),
        _ => format!("({}) 0xc000000000", t.name).into_bytes(),
    }
}

pub fn format_chain(p: &V) -> Vec<u8> {
    let mut b = Vec::new();
    let prev = panic_prev(p);
    if !prev.is_nil() {
        b.extend(format_chain(&prev));
        b.push(b'\t');
    }
    b.extend_from_slice(b"panic: ");
    b.extend(format_panic_value(&panic_value(p)));
    if panic_recovered(p) {
        b.extend_from_slice(b" [recovered]");
    }
    b.push(b'\n');
    b
}

pub fn report_panic(p: &V) -> ! {
    let _root = temp_root(&[p.clone()]);
    out::stderr(&format_chain(p));
    std::process::exit(2)
}

fn report_stats() {
    if std::env::var("GOALCHEMY_HEAP_STATS").is_ok() {
        let (peak, n) = heap_stats();
        out::stderr(format!("heap: peak {} live {} collections {}\n", peak, heap_live(), n).as_bytes());
    }
}

/// Runs body on a thread with a large stack, reporting unrecovered panics.
pub fn run_large(body: impl FnOnce() + Send + 'static) -> ! {
    std::panic::set_hook(Box::new(|info| {
        if info.payload().is::<GoPanicPayload>() {
            return;
        }
        let msg = info
            .payload()
            .downcast_ref::<String>()
            .cloned()
            .or_else(|| info.payload().downcast_ref::<&str>().map(|s| s.to_string()))
            .unwrap_or_default();
        out::stderr(format!("goalchemy runtime fault: {}\n", msg).as_bytes());
    }));
    let h = std::thread::Builder::new().stack_size(1 << 30).spawn(move || {
        body();
        report_stats();
    });
    match h.map(|h| h.join()) {
        Ok(Ok(())) => std::process::exit(0),
        _ => std::process::exit(101),
    }
}

/// The entry point of a sequential program.
pub fn program_main(nglobals: usize, init: fn(), entry: fn()) -> ! {
    run_large(move || {
        init_globals(nglobals);
        init();
        if let Err(p) = catch(entry) {
            report_panic(&p);
        }
    })
}
