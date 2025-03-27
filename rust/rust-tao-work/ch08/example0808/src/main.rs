fn main() {
    let str = "bards";
    let mut chars = str.chars();
    assert_eq!(Some('b'), chars.next());

    let mut bytes = str.bytes();
    assert_eq!(Some(98), bytes.next())
}
