use std::vec;

fn func_name(arg1: u32, arg2: String) -> Vec<u32> {
    // 函数体
    vec![1]
}

// 使用
fn r#match(needle: &str, haystack: &str) -> bool {
    haystack.contains(needle)
}
fn main() {
    assert_eq!(func_name(3, "fo".to_string()), vec![1]);
    assert!(r#match("foo", "foobar"));
}
