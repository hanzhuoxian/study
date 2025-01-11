fn main() {
    assert_eq!(std::mem::size_of::<&[u32; 5]>(), 8);
    assert_eq!(std::mem::size_of::<&[u32]>(), 16);
}
