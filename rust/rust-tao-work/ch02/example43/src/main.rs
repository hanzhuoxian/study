fn main() {
    let s: &Option<String> = &Some("hello".to_string());
    match s {
        &Some(ref s) => println!("s is {}", s),
        _ => (),
    }

    match s {
        Some(s) => println!("s is {}", s),
        _ => (),
    }
}
