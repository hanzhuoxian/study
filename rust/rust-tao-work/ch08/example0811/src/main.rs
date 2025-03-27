fn main() {
    let mut hello  =  String::from("Hello, ");
    hello.push('R');
    hello.push_str("ust");
    assert_eq!(hello, "Hello, Rust");
}
