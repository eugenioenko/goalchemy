use goalchemy_generated::*;

fn options() -> CallOptions { CallOptions::default() }

fn main() {
    for value in [1.25, -0.0, f64::INFINITY, f64::NEG_INFINITY, f64::NAN] {
        let small = Scalar32(value as f32, options()).wait().unwrap();
        let wide = Scalar64(value, options()).wait().unwrap();
        assert!(if value.is_nan() { small.is_nan() } else { small.to_bits() == (value as f32).to_bits() }, "float32 IEEE scalar");
        assert!(if value.is_nan() { wide.is_nan() } else { wide.to_bits() == value.to_bits() }, "float64 IEEE scalar");
    }
    let both = Both(1.25, -0.0, options()).wait().unwrap();
    assert!(both.0 == 1.25 && both.1.is_sign_negative(), "multiple float results");
    let input = Value { Small: 1.25, Wide: -0.0, Values: vec![2.5], Nested: vec![vec![4.5]], Pair: vec![6.5, 7.5] };
    let mut first = Echo(input.clone(), options()).wait().unwrap();
    assert!(first.Small == 1.25 && first.Wide.is_sign_negative() && first.Values[0] == 3.5 && first.Nested[0][0] == 6.5 && first.Pair[1] == 7.5, "nested conversion");
    assert!(input.Values[0] == 2.5 && input.Nested[0][0] == 4.5, "input ownership");
    first.Values[0] = 99.0; first.Nested[0][0] = 99.0;
    let second = Echo(input.clone(), options()).wait().unwrap();
    assert!(second.Values[0] == 3.5 && second.Nested[0][0] == 6.5, "result ownership");
    assert_eq!(Suspended(input.clone(), options()).wait().unwrap().Values[0], 2.5, "suspension");
    let mut owned = Fixed(options()).wait().unwrap(); owned.Values[0] = 99.0;
    assert!(Fixed(options()).wait().unwrap().Values[0] == 1.5 && owned.Values[0] == 99.0, "global result ownership");
    let mut invalid = input; invalid.Pair = vec![1.0];
    assert_eq!(Echo(invalid, options()).wait().unwrap_err().kind, ErrorKind::InvalidArgument, "array length validation");
    println!("PASS float library");
}
