fn counter(i: i32) -> fn(i32) -> i32 {
    fn inc(n: i32) -> i32 {
        n + i
    }
    inc
}
fn main() {
    let f = counter(5);
    assert_eq!(2, f(1));
}
