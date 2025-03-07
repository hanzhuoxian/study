#[derive(Debug)]
struct S {
    i: i32,
}

fn f(ref _s: S) {
    println!("{:p}", _s);
}
fn main() {
    let s = S { i: 42 };
    f(s);
    // println!("{:?}", s)
}
