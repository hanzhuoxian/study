use std::ops::Mul;

fn square<T: Mul<T, Output = T>>(x: T, y: T) -> T {
    x * y
}

fn main() {
    let a: i32 = square(37, 41);
    let b: f64 = square(37.2, 41.1);
    assert_eq!(a, 1517);
    assert_eq!(b, 1528.92);
    let a = square::<u32>(37, 41);
    let b= square::<f64>(37.2, 41.1);
    assert_eq!(a, 1517);
    assert_eq!(b, 1528.92);
}
