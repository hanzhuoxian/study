fn main() {
    let string: String = String::new();
    assert_eq!(string, "");
    let string = String::from("hello");
    assert_eq!(string, "hello");
    let string = String::with_capacity(10);
    assert_eq!(string, "");

    let str: &'static str = "the world of hello";
    let string: String = str.chars().filter(|c| !c.is_whitespace()).collect();
    assert_eq!(string, "theworldofhello");

    let string = str.to_owned();
    assert_eq!(string, "the world of hello");

    let string = str.to_string();
    assert_eq!(string, "the world of hello");

    let str: &str =  &string[13..18];
    assert_eq!(str, "hello");

}