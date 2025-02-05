fn foo<'a>() -> &'a str{
    let a = "hello".to_string();
    &a
}
fn main() {
    let x = foo();
}
