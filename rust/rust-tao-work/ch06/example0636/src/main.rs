fn main() {
    let s = "hello".to_string();
    let c = move || s;
    c();
    c();
}
