fn foo<'a>(a: &'a str, b: &'a str) -> &'a str {
    let result = String::from("relay long string");
    result.as_str() // 标注了输出的生命周期也没用，返回去会造成垂悬指针。
}
fn main() {
    let x = "hello";
    let y = "rust";
    foo(x, y);
}
