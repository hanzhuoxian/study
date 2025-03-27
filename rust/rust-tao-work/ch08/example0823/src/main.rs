fn main() {
    let s = " Hello\tWorld\t";
    assert_eq!("Hello\tWorld", s.trim());
    assert_eq!("Hello\tWorld\t", s.trim_start());
    assert_eq!(" Hello\tWorld", s.trim_end());
}
