use super::*;
pub fn lib_os_write_file(name: V, data: V, perm: V) -> V {
    let r = (|| -> Result<(), i64> {
        use std::io::Write;
        let path = os_path(&name.bytes())?;
        let (h, o, n, _, _) = slice_parts(&data);
        if n as u64 > OS_MAX_FILE {
            return Err(6);
        }
        let body = if h == 0 {
            vec![]
        } else {
            byte_snapshot(h, o as usize, n as usize)
        };
        let mut options = std::fs::OpenOptions::new();
        options.write(true).create(true).truncate(true);
        #[cfg(unix)]
        {
            use std::os::unix::fs::OpenOptionsExt;
            options.mode((perm.i() & 0o777) as u32);
        }
        let mut f = options.open(path).map_err(|e| os_status(&e))?;
        f.write_all(&body).map_err(|e| os_status(&e))
    })();
    V::Int(match r {
        Ok(()) => 0,
        Err(code) => code,
    })
}
