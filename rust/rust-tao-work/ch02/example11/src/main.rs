fn is_true() -> bool {
    true
}
fn true_marker() -> fn() -> bool {
    is_true
}
fn main() {
    assert_eq!(true_marker()(), true)
}
