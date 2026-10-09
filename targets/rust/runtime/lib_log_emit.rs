use super::*;
pub fn lib_log_emit(level: V, unix_nano: V, message: V, attrs: V, text: V) {
    let (h, o, n, _, _) = slice_parts(&attrs);
    let pairs: Vec<Rc<[u8]>> = (0..n as usize)
        .map(|i| slot(h, o as usize + i).bytes())
        .collect();
    log_emit(
        level.i(),
        unix_nano.i(),
        &message.bytes(),
        &pairs,
        &text.bytes(),
    );
}
