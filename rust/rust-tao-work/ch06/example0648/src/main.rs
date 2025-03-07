fn square() -> Box<dyn FnOnce(i32) -> i32> {
    Box::new(|i| i * i)
}
fn main() {
    let square = square();
    assert_eq!(4, square(2));
}
