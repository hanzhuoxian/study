fn main() {
    let four: u32 = "4".parse().unwrap();
    assert_eq!(four, 4);
    let four = "4".parse::<u32>();
    assert_eq!(four, Ok(4));
}
