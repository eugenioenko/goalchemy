//! std.errors.new: a distinct *errors.errorString per call.
use super::*;

fn error_string_text(_: &[V], a: Vec<V>) -> V {
    slot(a[0].h(), 0)
}

fn eq_identity(a: &V, b: &V) -> bool {
    veq(a, b)
}

pub static ERRORS_ERROR_STRING: TypeDesc = TypeDesc {
    id: 0x7fff_0010,
    name: "*errors.errorString",
    kind: "pointer",
    eq: eq_identity,
    key: key_basic,
    methods: &[("Error", -1, error_string_text)],
    basic: "",
    comparable: true,
};

pub fn std_errors_new(text: V) -> V {
    boxv(&ERRORS_ERROR_STRING, vals(vec![text]))
}
