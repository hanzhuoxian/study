use std::rc::Rc;


fn main() {
    let x = Rc::new("hello");
    let y = x.clone();
    let z = *x;
    println!("{:?} {:?} {:?}", x, y, z);
}
