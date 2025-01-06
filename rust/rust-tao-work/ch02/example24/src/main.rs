fn main() {
    let x = true;
    println!("{}", x);
    let y: bool = false;
    assert_eq!(x as i32, 1);
    assert_eq!(y as i32, 0);
    let x: u8 = 255;
    println!("{x}")
}
