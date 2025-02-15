#[derive(Debug)]
struct A {
    #[allow(dead_code)]
    a: i32,
    #[allow(dead_code)]
    b: u32,
}
fn main() {
    let a = A { a: 1, b: 2 };
    let b = a;
    // println!("{:?}", a); // borrow of moved value: `a` value borrowed here after move
    println!("{:?}", b);
}
