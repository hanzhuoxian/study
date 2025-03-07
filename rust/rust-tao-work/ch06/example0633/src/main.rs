#![feature(fn_traits)]
fn main() {
    let s = "hello".to_string();
    let c = || s;
    c();
    c.call();
    // c();
    // println!("{:?}", s);
}
