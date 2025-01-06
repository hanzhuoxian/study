use std::fmt::Debug;

fn match_option<T: Debug>(o: Option<T>) {
    match o {
        Some(i) => println!("{:?}", i),
        None => println!("nothing"),
    }
}

fn main() {
    let a: Option<i32> = Some(3);
    match_option(a);
    let b: Option<&str> = Some("hello");
    match_option(b);
    let c: Option<char> = Some('A');
    match_option(c);
    let mut d: Option<u32> = None;
    match_option(d);
    d = Some(3);
    match_option(d);
}
