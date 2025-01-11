fn sum<T: std::ops::Add<T, Output = T>>(a: T, b: T) -> T {
    a + b
}
fn main() {
    println!("{:?}", sum(1, 2));
}
