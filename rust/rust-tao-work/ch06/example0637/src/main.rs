fn main() {
    let mut x = 0;
    let mut incr_x = || x += 1;
    incr_x();
    println!("{:?}", x);

    let mut x = 0;
    let mut incr_x = move || x += 1;
    println!("{:?}", incr_x());
}
