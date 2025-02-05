#[allow(dead_code)]
struct A {
    a: u32,
    b: Box<u64>,
}

#[allow(dead_code)]
struct B(i32, u64, char);
struct N;

#[allow(dead_code)]
enum E {
    H(u32),
    M(Box<u32>),
}
#[allow(dead_code)]
union U {
    u: u32,
    v: u64,
}
fn main() {
    println!("Box<u32>: {:?}", std::mem::size_of::<Box<u32>>());
    println!("A:{:?}", size_of::<A>());
    println!("B:{:?}", size_of::<B>());
    println!("N:{:?}", size_of::<N>());
    println!("E:{:?}", size_of::<E>());
    println!("U:{:?}", size_of::<U>());
}
