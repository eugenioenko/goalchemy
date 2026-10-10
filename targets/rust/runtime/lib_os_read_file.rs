use super::*;
pub const OS_MAX_FILE: u64 = 1 << 30;
pub fn os_path(name: &[u8]) -> Result<std::path::PathBuf, i64> {
    if name.is_empty() {
        return Err(1);
    }
    if name.contains(&0) {
        return Err(8);
    }
    #[cfg(unix)]
    {
        use std::os::unix::ffi::OsStrExt;
        Ok(std::path::PathBuf::from(std::ffi::OsStr::from_bytes(name)))
    }
    #[cfg(not(unix))]
    {
        Ok(std::path::PathBuf::from(
            String::from_utf8_lossy(name).into_owned(),
        ))
    }
}
pub fn os_status(e: &std::io::Error) -> i64 {
    use std::io::ErrorKind::*;
    match e.kind() {
        NotFound => 1,
        AlreadyExists => 2,
        PermissionDenied => 3,
        IsADirectory => 4,
        NotADirectory => 5,
        FileTooLarge => 6,
        Unsupported => 7,
        InvalidInput => 8,
        _ => 9,
    }
}
pub fn lib_os_read_file(name: V) -> V {
    let r = (|| -> Result<Vec<u8>, i64> {
        use std::io::Read;
        let mut f = std::fs::File::open(os_path(&name.bytes())?).map_err(|e| os_status(&e))?;
        if f.metadata().map(|m| m.len()).unwrap_or(0) > OS_MAX_FILE {
            return Err(6);
        }
        let mut data = Vec::new();
        f.by_ref()
            .take(OS_MAX_FILE + 1)
            .read_to_end(&mut data)
            .map_err(|e| os_status(&e))?;
        if data.len() as u64 > OS_MAX_FILE {
            return Err(6);
        }
        Ok(data)
    })();
    match r {
        Ok(data) => {
            let n = data.len() as u32;
            tuple(vec![V::ByteSlice(byte_array(data).h(), 0, n, n), V::Int(0)])
        }
        Err(code) => tuple(vec![BYTE_NIL, V::Int(code)]),
    }
}
