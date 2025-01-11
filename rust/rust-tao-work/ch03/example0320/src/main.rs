fn foo<T>(x: T) -> T {
    return x;
}

fn foo_1(x: i32) -> i32 {
    return x;
}
fn foo_2(x: &'static str) -> &'static str {
    return x;
}

fn main() {
    assert_eq!(foo_1(1), foo(1));
    assert_eq!(foo_2("hello"), foo("hello"));
}
