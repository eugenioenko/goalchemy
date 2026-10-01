//! core.print: print and println of Booleans, integers, and strings to
//! standard error. Unsigned 64-bit operands are formatted by the caller.
use super::*;

pub fn print_bytes(v: &V) -> Vec<u8> {
    match v {
        V::Str(b) => b.to_vec(),
        V::Bool(b) => if *b { b"true".to_vec() } else { b"false".to_vec() },
        V::Int(i) => i.to_string().into_bytes(),
        V::Nil => b"nil".to_vec(),
        _ => b"?".to_vec(),
    }
}

pub fn print_string(v: V, unsigned: bool) -> V {
    if unsigned {
        return u64s(v);
    }
    s(&print_bytes(&v))
}

pub fn go_print(args: Vec<V>, newline: bool) {
    let mut b = Vec::new();
    for (i, a) in args.iter().enumerate() {
        if newline && i > 0 {
            b.push(b' ');
        }
        b.extend(print_bytes(a));
    }
    if newline {
        b.push(b'\n');
    }
    out::stderr(&b);
}
