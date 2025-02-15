fn return_str<'a>() -> &'a str {
    let mut s = "Rust".to_string();
    for i in 1..3 {
        s.push_str("Good");
    }
    &s // 如果返回会造成垂悬指针。
}
fn main() {
}
