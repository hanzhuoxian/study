fn main() {
    let mut s = String::with_capacity(3);
    s.insert(0, ' ');
    s.insert(1, 'f');
    s.insert(2, 'o');
    s.insert(3, 'o');
    s.insert_str(0, "bar");
    assert_eq!("bar foo", s);
}
