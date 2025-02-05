struct A {
    a: u8,
    b: u32,
    c: u16,
}
fn main() {
    println!("{}", std::mem::size_of::<A>());
}
