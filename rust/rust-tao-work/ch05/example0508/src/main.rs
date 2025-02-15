fn main() {
    let a = ("a".to_string(), "b".to_string());
    let b = a;
    // println!("{:?}", a);// borrow of moved value: `a` value borrowed here after move
    println!("{:?}", b);
    let c = (1, 2, 3);
    let d = c;
    println!("{:?} {:?}", c, d);
}
