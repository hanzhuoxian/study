use std::cell::Cell;

struct Foo {
    x: u32,
    y: Cell<u32>,
}
fn main() {
    let foo = Foo {
        x: 1,
        y: Cell::new(3),
    };
    assert_eq!(foo.x, 1);
    assert_eq!(foo.y.get(), 3);
    foo.y.set(5);
    assert_eq!(foo.y.get(), 5);
}
