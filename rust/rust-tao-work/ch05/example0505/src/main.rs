fn main() {
    let x = Box::new(5);
    let y = x;
    // println!("{:?}", x);// borrow of moved value: `x` value borrowed here after move
    println!("{:?}", y);
}
