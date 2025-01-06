fn main() {
    let boolean = true;
    let mut binary = 0;
    if let true = boolean {
        binary = 1
    }
    assert_eq!(binary, 1);
}
