fn main() {
    let a = Some("hello".to_string());
    match a {
        Some(s) => println!("{:?}", s),
        _ => println!("nothing"),
    }

    // println!("{:?}", a); // borrow of partially moved value: `a` partial move occurs because value has type `String`, which does not implement the `Copy` trait
}
