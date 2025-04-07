use std::panic;

fn sum(a: i32, b: i32) -> i32 {
    a + b
}
fn main() {
    assert!(panic::catch_unwind(|| { println!("Hello Rust") }).is_ok());

    assert!(panic::catch_unwind(|| { panic!("oh no") }).is_err());
}
