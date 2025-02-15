fn main() {
    let orig = Box::new(5);
    println!("{:?}", *orig);
    let stolen = orig; // orig move stolen
    println!("{:?}", *stolen);
    // println!("{:?}", *orig); // borrow of moved value: `orig` value borrowed here after move
}
