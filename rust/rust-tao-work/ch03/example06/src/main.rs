enum Void {}
struct Foo;
struct Baz {
    foo: Foo,
    qux: (),
    baz: [u8; 0],
}
fn main() {
    assert_eq!(std::mem::size_of::<()>(), 0);
    assert_eq!(std::mem::size_of::<Foo>(), 0);
    assert_eq!(std::mem::size_of::<Baz>(), 0);
    assert_eq!(std::mem::size_of::<Void>(), 0);
    assert_eq!(std::mem::size_of::<[(); 10]>(), 0);
    let baz = Baz {
        foo: Foo {},
        qux: (),
        baz: [],
    };
    assert_eq!(std::mem::size_of_val(&baz), 0);
    assert_eq!(std::mem::size_of_val(&baz.foo), 0);
    assert_eq!(std::mem::size_of_val(&baz.qux), 0);
    assert_eq!(std::mem::size_of_val(&baz.baz), 0);
}
