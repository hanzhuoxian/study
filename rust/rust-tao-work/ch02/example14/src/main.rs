fn closure_math<F: Fn() -> i32>(op: F) -> i32 {
    op()
}

fn closure_math_args<F: Fn(i32, i32) -> i32>(op: F, a: i32, b: i32) -> i32 {
    op(a, b)
}

fn main() {
    let a = 2;
    let b = 3;

    assert_eq!(closure_math(|| a + b), 5);
    assert_eq!(closure_math(|| a * b), 6);

    assert_eq!(
        closure_math_args(|c: i32, d: i32| -> i32 { a + b + c + d }, 2, 3),
        10
    )
}
