trait Add<RHS = Self> {
    type Output;
    fn add(self, other: u64) -> Self::Output;
}

impl Add<u64> for u32 {
    type Output = u64;
    fn add(self, other: u64) -> Self::Output {
        (self as u64) + other
    }
}
fn main() {
    let a: u32 = 1;
    let b: u64 = 2;
    println!("{}", a.add(b));
}
