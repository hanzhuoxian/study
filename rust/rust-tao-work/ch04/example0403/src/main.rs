union A {
    a: u32,
    b: f32,
    c: f64,
}
union B {
    a: u8,
}
fn main() {
    println!("{}", std::mem::size_of::<A>());
    println!("{}", std::mem::size_of::<B>());
}
