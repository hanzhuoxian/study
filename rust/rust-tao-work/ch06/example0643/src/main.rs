

fn call<F: Fn(i32) -> i32>(f: F) -> i32 {
    f(1)
}
fn counter(i: i32) -> i32 {
    i + 1
}

fn main() {
    let result = call(counter);
    assert_eq!(result, 2)
}
