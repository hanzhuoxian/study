fn main() {
    struct Foo<T>(T);
    struct Bar<T: ?Sized>(T);
    struct Baz<T: ?Copy>(T);

    let x = Foo(0);
    println!("x is {}", x.0);

    let y = Bar("hello");
    println!("y is {}", y.0);


}
