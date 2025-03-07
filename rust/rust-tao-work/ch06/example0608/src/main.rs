fn add_sub(x: i32, y: i32) -> (i32, i32) {
    (x + y, x - y)
}
fn main() {
    let (a, b) = add_sub(5, 8);
    println!("a: {a} b: {b}");
}
