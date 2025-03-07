fn counter(i: i32) -> Box<dyn Fn(i32) -> i32> {
    Box::new(move |n| n + i)
}
fn impl_counter(i: i32) -> impl Fn(i32) -> i32 {
    move |n|n+i
}
fn main() {
    let f = counter(5);
    assert_eq!(6, f(1));
    let f = impl_counter(5);
    assert_eq!(6, f(1));
}
