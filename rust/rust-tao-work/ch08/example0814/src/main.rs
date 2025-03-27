fn main() {
    let mut s = "Hello, ".to_string();
    s += "Rust";
    assert_eq!(s, "Hello, Rust");

    let left = "Hello".to_string();
    let right = "Rust".to_string();
    let sept = ", ".to_string();
    let s = left + &sept + &right;
}
