fn main() {
    let mut hello = "Hello, world!".to_string();
    hello.remove(3);
    assert_eq!(hello, "Helo, world!");
    assert_eq!(Some('!'), hello.pop());

    hello.truncate(4);
    assert_eq!(hello, "Helo");

    hello.clear();
    assert_eq!("", hello);

}
