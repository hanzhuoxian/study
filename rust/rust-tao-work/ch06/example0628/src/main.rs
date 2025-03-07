fn main() {
    let env_val = 1;
    let c = || env_val + 2;
    assert_eq!(c(), 3);
}
