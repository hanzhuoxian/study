fn join(s: &String) -> String {
    // let append = *s;
    // "Hello".to_string() + &append
    s.to_string()
}
fn main() {
    let x = "hello".to_string();
    join(&x);
}
