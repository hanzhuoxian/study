#[derive(Clone)]
struct A {
    a: i32,
    b: Box<i32>, // this field does not implement `Copy`
}
fn main() {
    let a = A {
        a: 3,
        b: Box::new(3),
    };
    let b = a.clone();
    println!("{:?} {:?} {:?} {:?}", a.a , a.b, b.a, b.b);
}
