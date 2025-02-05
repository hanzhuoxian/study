use std::u32;

fn main() {
    let a = u32::MAX;
    let b = a as u16;
    assert_eq!(b, 65535);

    let e = -1i32;
    let f = e as u32;
    println!("{:?}", e.abs()); // 1
    println!("{:?}", f);// 4294967295
}
