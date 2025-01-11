struct Point<T> {
    x: T,
    y: T,
}
fn main() {
    let p = Point { x: 1, y: 2 };
    assert_eq!(p.x, 1);
    assert_eq!(p.y, 2);
    let fp = Point { x: 1.1, y: 2f64 };
    assert_eq!(fp.x, 1.1);
    assert_eq!(fp.y, 2.0);
}
