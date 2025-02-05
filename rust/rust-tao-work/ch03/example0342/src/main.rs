use std::ops::Add;

fn sum<T>(a: impl Add<Output = T>, b: impl Add<Output = T>) -> T {
    a + b
}

fn main() {
    println!("Hello, world!");
}
