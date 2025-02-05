fn main() {
    let x = 42;
    let y = Box::new(5);
    println!("{:p} {:p}", &x, y);
    let x2 = x;
    let y2 = y;
    // println!("{:p}", y);
    println!("{:?}", x);
    println!("{:p} {:p}", &x2, y2);
}
