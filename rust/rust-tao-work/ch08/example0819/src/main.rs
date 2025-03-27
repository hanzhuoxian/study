fn main() {
    let s = String::from("hello, 世界");
    let i = s.find("世界").unwrap();
    assert_eq!(i, 7);
    let i = s.find("界").unwrap();
    assert_eq!(i, 10);
    
}
