#[derive(Debug)]
struct S {
    i: i32,
}

fn f(ref _s: S) {
    println!("{:p}", _s);
}
fn foo(_: S) {}
fn main() {
    let s = S { i: 42 };
    f(s);
    // println!("{:?}", s)

    let s1 = S { i: 32 };
    foo(s1);
}
