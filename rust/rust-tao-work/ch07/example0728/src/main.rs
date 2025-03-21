#[derive(Debug)]
struct PrintDrop(&'static str);

impl Drop for PrintDrop {
    fn drop(&mut self) {
        println!("Dropping {}", self.0)
    }
}
#[derive(Debug)]
enum E {
    Foo(PrintDrop, PrintDrop),
}

#[derive(Debug)]
struct Foo {
    x: PrintDrop,
    y: PrintDrop,
    z: PrintDrop,
}
fn main() {
    let e = E::Foo(PrintDrop("a"), PrintDrop("b"));
    let f = Foo {
        x: PrintDrop("x"),
        y: PrintDrop("y"),
        z: PrintDrop("z"),
    };

    println!("{:?}{:?}", e, f);
}
