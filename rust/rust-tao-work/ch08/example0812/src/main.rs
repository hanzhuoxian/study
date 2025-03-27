fn main() {
    let mut hello = "Hello".to_string();
    hello.extend([',', ' ', 'R', 'u'].iter());
    hello.extend("st".chars());
    assert_eq!(hello, "Hello, Rust");
}
