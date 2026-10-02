mod rt;
use rt::*;

fn native(x: &V) -> Vec<u8> {
    let (h, o, l, c, bytes) = slice_parts(x);
    assert!(bytes);
    assert!(o <= u32::MAX - c && l <= c);
    with(h, |obj| match obj {
        Obj::Bytes(v) => {
            assert_eq!(std::mem::size_of::<u8>(), 1);
            assert!(v.len() >= (o + c) as usize);
            v.clone()
        }
        _ => panic!("non-native byte backing"),
    })
}
fn view(x: &V, lo: i64, hi: i64, max: i64) -> V {
    reslice(x.clone(), int(lo), int(hi), int(max), false)
}
fn expect_panic(f: impl FnOnce()) {
    let p = match catch(f) { Err(p) => p, Ok(_) => panic!("expected source panic") };
    assert!(std::ptr::eq(dyn_type(&panic_value(&p)).unwrap(), &RUNTIME_ERROR));
}
fn expect_fault(f: impl FnOnce()) {
    let err = std::panic::catch_unwind(std::panic::AssertUnwindSafe(f)).expect_err("expected implementation fault");
    let msg = err.downcast_ref::<String>().map(|s| s.as_str()).or_else(|| err.downcast_ref::<&str>().copied()).unwrap_or("");
    assert!(msg.starts_with("goalchemy fault:"), "unexpected host unwind: {}", msg);
}

#[test]
fn nil_empty_and_capacity() {
    for x in [BYTE_NIL, make_byte_slice(int(0), int(0))] {
        let nil = nil_slice(&x);
        for y in [BYTE_NIL, to_bytes(s(b""))] {
            assert_eq!(copy(x.clone(), y.clone(), None).i(), 0);
            assert_eq!(copy(y.clone(), x.clone(), None).i(), 0);
            assert_eq!(slice_parts(&append_slice(x.clone(), y, None)), slice_parts(&x));
        }
        clear_slice(x.clone(), || int(99));
        assert_eq!(copy_string(x.clone(), s(b"abc")).i(), 0);
        assert_eq!(copy_string(x.clone(), s(b"")).i(), 0);
        assert_eq!(slice_parts(&append_string(x.clone(), s(b""))), slice_parts(&x));
        assert_eq!(slice_parts(&append_bytes(x.clone(), &[])), slice_parts(&x));
        let r = view(&x, 0, 0, 0);
        assert_eq!(nil_slice(&r), nil);
        assert_eq!(native(&append_bytes(r, &[255, 128])), vec![255, 128]);
    }
    assert!(nil_slice(&zero_byte_slice()));
    assert!(!nil_slice(&make_byte_slice(int(0), int(0))));
    assert_eq!(vals_len(slice_to_array(BYTE_NIL, 0, None).h()), 0);
    let x = make_byte_slice(int(2), int(6));
    assert_eq!(native(&x), vec![0; 6]);
    sset(x.clone(), int(0), int(-1));
    sset(x.clone(), int(1), int(384));
    assert_eq!(sget(x.clone(), int(0)).i(), 255);
    assert_eq!(sgetu(x.clone(), int(1)).i(), 128);
    let v = view(&x, 1, 2, 5);
    let a = append_bytes(v.clone(), &[3, 4]);
    assert_eq!(slice_parts(&v), (slice_parts(&x).0, 1, 1, 4, true));
    assert_eq!(slice_parts(&a), (slice_parts(&x).0, 1, 3, 4, true));
    assert_eq!(native(&x), vec![255, 128, 3, 4, 0, 0]);
    let grown = append_slice(a.clone(), a.clone(), None);
    assert_ne!(slice_parts(&grown).0, slice_parts(&a).0);
    assert_eq!(native(&grown), vec![128, 3, 4, 128, 3, 4, 0, 0]);
    assert_eq!(slice_parts(&grown).3, 8);
    sset(grown, int(0), int(9));
    assert_eq!(sget(a, int(0)).i(), 128);
    clear_slice(view(&x, 1, 1, 6), zero_int);
    clear_slice(view(&x, 6, 6, 6), zero_int);
    clear_slice(view(&x, 1, 3, 6), || int(99));
    assert_eq!(native(&x), vec![255, 0, 0, 4, 0, 0]);
}

#[test]
fn overlap_binary_and_array_helpers() {
    let x = to_bytes(s(&[1, 2, 3, 4, 5, 6]));
    assert_eq!(copy(view(&x, 1, 6, 6), x.clone(), None).i(), 5);
    assert_eq!(native(&x), vec![1, 1, 2, 3, 4, 5]);
    assert_eq!(copy(x.clone(), view(&x, 2, 6, 6), None).i(), 4);
    assert_eq!(native(&x), vec![2, 3, 4, 5, 4, 5]);
    for (dest, src) in [(0, 1), (1, 0)] {
        let x = to_bytes(s(&[1, 2, 3, 4, 5, 6]));
        let r = append_slice(view(&x, dest, dest + 2, 6), view(&x, src, src + 3, 6), None);
        assert_eq!(slice_parts(&r).0, slice_parts(&x).0);
        let expected = if dest == 0 { vec![1, 2, 2, 3, 4, 6] } else { vec![1, 2, 3, 1, 2, 3] };
        assert_eq!(native(&x), expected);
    }
    let binary = [0, 255, 128, 192, 175, 65];
    let x = to_bytes(s(&binary));
    let saved = from_bytes(x.clone());
    sset(x.clone(), int(0), int(44));
    assert_eq!(&saved.bytes()[..], &binary);
    let y = to_bytes(saved.clone());
    sset(y, int(1), int(1));
    assert_eq!(&saved.bytes()[..], &binary);
    let x = append_string(view(&x, 0, 1, 1), saved.clone());
    assert_eq!(native(&x), vec![44, 0, 255, 128, 192, 175, 65]);
    assert_eq!(copy_string(view(&x, 2, 7, 7), saved).i(), 5);
    assert_eq!(native(&x), vec![44, 0, 0, 255, 128, 192, 175]);
    assert!(from_bytes(BYTE_NIL).bytes().is_empty());
    let a = byte_array(binary.to_vec());
    let b = byte_array_clone(&a);
    let alias = slice_array(a.clone(), V::Nil, V::Nil, V::Nil, false);
    let key = byte_array_key(&a);
    assert!(byte_array_eq(&a, &b));
    set_slot(a.h(), 0, int(7));
    assert_eq!(sget(alias.clone(), int(0)).i(), 7);
    assert!(!byte_array_eq(&a, &b));
    assert!(key == byte_array_key(&b));
    byte_array_set(&a, &b);
    assert_eq!(sget(alias.clone(), int(0)).i(), 0);
    byte_array_set(&a, &a);
    let detached = slice_to_array(alias.clone(), 6, None);
    set_slot(a.h(), 0, int(9));
    assert_eq!(slot(detached.h(), 0).i(), 0);
    let empty = byte_array(vec![]);
    assert!(!nil_slice(&slice_array(empty.clone(), V::Nil, V::Nil, V::Nil, false)));
    assert!(byte_array_eq(&empty, &byte_array_clone(&empty)));
}

#[test]
fn validation_before_write_and_generic_consumers() {
    let x = to_bytes(s(&[255, 128]));
    for i in [int(-1), int(2), int(i64::MIN), int(-1)] {
        expect_panic(|| { sset(x.clone(), i.clone(), int(1)); });
        expect_panic(|| { ssetu(x.clone(), i.clone(), int(1)); });
        expect_panic(|| { asetu(&byte_array(vec![1, 2]), &i, 2, int(1)); });
        assert_eq!(native(&x), vec![255, 128]);
    }
    expect_panic(|| { make_byte_slice(int(-1), int(2)); });
    expect_panic(|| { make_byte_slice(int(2), int(1)); });
    expect_panic(|| { make_byte_slice(int(i64::MIN), int(i64::MIN)); });
    expect_fault(|| { make_byte_slice(int(0), int(u32::MAX as i64)); });
    expect_panic(|| { reslice(x.clone(), int(0), int(i64::MIN), V::Nil, true); });
    // A malformed near-limit header is never indexed: append validates its new
    // capacity before touching the real backing or narrowing length to u32.
    let bad = V::ByteSlice(slice_parts(&x).0, 0, u32::MAX / 2, u32::MAX / 2);
    expect_fault(|| { append_bytes(bad, &[1]); });
    assert_eq!(native(&x), vec![255, 128]);
    let ints = make_slice(int(1), int(3), || int(1000), false);
    sset(ints.clone(), int(0), int(-1));
    let ints = append(ints, vec![int(500)], None);
    assert_eq!(sget(ints.clone(), int(0)).i(), -1);
    assert_eq!(sget(ints.clone(), int(1)).i(), 500);
    assert_eq!(sget(view(&ints, 0, 3, 3), int(2)).i(), 1000);
    let rune = to_runes(s("é".as_bytes()));
    assert_eq!(sget(rune.clone(), int(0)).i(), 233);
    assert_eq!(from_runes(rune).bytes().as_ref(), "é".as_bytes());
}

static BYTE_SLICE_TYPE: TypeDesc = TypeDesc {
    id: 1, name: "[]uint8", kind: "slice", basic: "", comparable: false,
    eq: |_, _| eq_uncomparable("[]uint8"),
    key: |_| key_unhashable("[]uint8"), methods: &[],
};

#[test]
fn byte_leaf_gc_survives_every_root_shape() {
    // Distinct allocations ensure one working root shape cannot mask another.
    let f = Fr::new(7);
    f.s(0, cellv(to_bytes(s(&[255, 128, 3])))); // source pointer to a slice cell
    let field_array = byte_array(vec![7, 8, 9]);
    f.s(1, vals(vec![slice_array(field_array, int(1), int(3), V::Nil, false)]));
    f.s(2, boxv(&BYTE_SLICE_TYPE, to_bytes(s(&[11, 12]))));
    f.s(3, func(-1, |e, _| e[0].clone(), vec![to_bytes(s(&[21, 22]))]));
    let clone_source = byte_array(vec![31, 32]);
    f.s(4, byte_array_clone(&clone_source));
    f.s(5, to_bytes(s(&[41, 42]))); // only a direct ByteSlice retains this handle
    f.s(6, byte_array(vec![51, 52])); // only a direct Obj retains this array
    for _ in 0..12 {
        byte_array(vec![0; 16]);
        safepoint();
        assert_eq!(sget(pget(&f.g(0)), int(0)).i(), 255);
        assert_eq!(sget(fld(&f.g(1), 0), int(0)).i(), 8);
        assert_eq!(sget(unboxed(&f.g(2)), int(1)).i(), 12);
        assert_eq!(sget(callv(&f.g(3), vec![]), int(0)).i(), 21);
        assert_eq!(slot(f.g(4).h(), 0).i(), 31);
        assert_eq!(sget(f.g(5), int(0)).i(), 41);
        assert_eq!(slot(f.g(6).h(), 0).i(), 51);
    }
    assert!(heap_stats().1 > 0);
}
