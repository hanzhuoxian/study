trait Foo {}
struct Bar<'a> {
    x: &'a i32,
}

impl<'a> Foo for Bar<'a> {}
fn main() {
    let num = 5;
    let box_bar = Box::new(Bar { x: &num });
    let obj = box_bar as Box<dyn Foo>;
}
