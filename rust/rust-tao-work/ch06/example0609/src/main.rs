fn gcd(a: u32, b: u32) -> u32 {
    if b == 0 {
        return a;
    }
    return gcd(b, a % b);
}
fn main() {
    let g = gcd(60, 40);
    assert_eq!(g, 20)
}
