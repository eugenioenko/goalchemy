//! Minimal JSON values; numbers keep their source text.

#[derive(Clone, Debug)]
pub enum J {
    Null,
    Bool(bool),
    Num(String),
    Str(String),
    Arr(Vec<J>),
    Obj(Vec<(String, J)>),
}

impl J {
    pub fn get(&self, k: &str) -> Option<&J> {
        match self {
            J::Obj(m) => m.iter().find(|(n, _)| n == k).map(|(_, v)| v),
            _ => None,
        }
    }

    pub fn str(&self) -> &str {
        match self {
            J::Str(s) | J::Num(s) => s,
            _ => "",
        }
    }

    pub fn arr(&self) -> &[J] {
        match self {
            J::Arr(a) => a,
            _ => &[],
        }
    }

    pub fn truthy(&self) -> bool {
        matches!(self, J::Bool(true))
    }

    pub fn obj(kv: Vec<(&str, J)>) -> J {
        J::Obj(kv.into_iter().map(|(k, v)| (k.to_string(), v)).collect())
    }
}

pub fn parse(s: &str) -> Result<J, String> {
    let b = s.as_bytes();
    let mut i = 0;
    let v = value(b, &mut i)?;
    ws(b, &mut i);
    if i != b.len() {
        return Err("trailing data".into());
    }
    Ok(v)
}

fn ws(b: &[u8], i: &mut usize) {
    while *i < b.len() && (b[*i] as char).is_ascii_whitespace() {
        *i += 1;
    }
}

fn value(b: &[u8], i: &mut usize) -> Result<J, String> {
    ws(b, i);
    if *i >= b.len() {
        return Err("unexpected end".into());
    }
    match b[*i] {
        b'{' => {
            *i += 1;
            let mut m = Vec::new();
            ws(b, i);
            if b.get(*i) == Some(&b'}') {
                *i += 1;
                return Ok(J::Obj(m));
            }
            loop {
                ws(b, i);
                let k = string(b, i)?;
                ws(b, i);
                expect(b, i, b':')?;
                let v = value(b, i)?;
                m.push((k, v));
                ws(b, i);
                if b.get(*i) == Some(&b',') {
                    *i += 1;
                    continue;
                }
                expect(b, i, b'}')?;
                return Ok(J::Obj(m));
            }
        }
        b'[' => {
            *i += 1;
            let mut a = Vec::new();
            ws(b, i);
            if b.get(*i) == Some(&b']') {
                *i += 1;
                return Ok(J::Arr(a));
            }
            loop {
                a.push(value(b, i)?);
                ws(b, i);
                if b.get(*i) == Some(&b',') {
                    *i += 1;
                    continue;
                }
                expect(b, i, b']')?;
                return Ok(J::Arr(a));
            }
        }
        b'"' => Ok(J::Str(string(b, i)?)),
        b't' => {
            *i += 4;
            Ok(J::Bool(true))
        }
        b'f' => {
            *i += 5;
            Ok(J::Bool(false))
        }
        b'n' => {
            *i += 4;
            Ok(J::Null)
        }
        _ => {
            let st = *i;
            while *i < b.len() && b"+-0123456789.eE".contains(&b[*i]) {
                *i += 1;
            }
            if st == *i {
                return Err("unexpected character".into());
            }
            Ok(J::Num(String::from_utf8_lossy(&b[st..*i]).into_owned()))
        }
    }
}

fn expect(b: &[u8], i: &mut usize, c: u8) -> Result<(), String> {
    if b.get(*i) != Some(&c) {
        return Err(format!("expected {}", c as char));
    }
    *i += 1;
    Ok(())
}

fn string(b: &[u8], i: &mut usize) -> Result<String, String> {
    expect(b, i, b'"')?;
    let mut out: Vec<u16> = Vec::new();
    let mut bytes: Vec<u8> = Vec::new();
    loop {
        let c = *b.get(*i).ok_or("unterminated string")?;
        *i += 1;
        match c {
            b'"' => {
                flush(&mut bytes, &mut out);
                return Ok(String::from_utf16_lossy(&out));
            }
            b'\\' => {
                flush(&mut bytes, &mut out);
                let e = b[*i];
                *i += 1;
                match e {
                    b'n' => out.push(10),
                    b't' => out.push(9),
                    b'r' => out.push(13),
                    b'b' => out.push(8),
                    b'f' => out.push(12),
                    b'u' => {
                        let h = std::str::from_utf8(&b[*i..*i + 4]).map_err(|e| e.to_string())?;
                        out.push(u16::from_str_radix(h, 16).map_err(|e| e.to_string())?);
                        *i += 4;
                    }
                    c => out.push(c as u16),
                }
            }
            c => bytes.push(c),
        }
    }
}

fn flush(bytes: &mut Vec<u8>, out: &mut Vec<u16>) {
    if !bytes.is_empty() {
        out.extend(String::from_utf8_lossy(bytes).encode_utf16());
        bytes.clear();
    }
}

pub fn write(v: &J, out: &mut String) {
    match v {
        J::Null => out.push_str("null"),
        J::Bool(b) => out.push_str(if *b { "true" } else { "false" }),
        J::Num(n) => out.push_str(n),
        J::Str(s) => {
            out.push('"');
            for c in s.chars() {
                match c {
                    '"' => out.push_str("\\\""),
                    '\\' => out.push_str("\\\\"),
                    '\n' => out.push_str("\\n"),
                    '\r' => out.push_str("\\r"),
                    '\t' => out.push_str("\\t"),
                    c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
                    c => out.push(c),
                }
            }
            out.push('"');
        }
        J::Arr(a) => {
            out.push('[');
            for (k, x) in a.iter().enumerate() {
                if k > 0 {
                    out.push(',');
                }
                write(x, out);
            }
            out.push(']');
        }
        J::Obj(m) => {
            out.push('{');
            for (k, (n, x)) in m.iter().enumerate() {
                if k > 0 {
                    out.push(',');
                }
                write(&J::Str(n.clone()), out);
                out.push(':');
                write(x, out);
            }
            out.push('}');
        }
    }
}
