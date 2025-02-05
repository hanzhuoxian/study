use std::rc::Rc;

fn main() {
    let x = Rc::new("hello");
    println!("{:?}", x.chars());
}
