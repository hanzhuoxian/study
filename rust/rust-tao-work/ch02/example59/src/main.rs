fn main() {
    println!("{}", 2); // nothing 代表 Display
    println!("{:?}", 2); // ? 代表 Debug
    println!("{:b}", 2); // b 代表 2 进制
    println!("{:o}", 2); // o 代表 8 进制
    println!("{:x}", 266); // x 代表 16 进制小写
    println!("{:X}", 266); // X 代表 16 进制大写
    println!("{:p}", &2);// p 代表指针
    println!("{:e}", 1000); // e 代表指数小写
    println!("{:E}", 1000); // E 代表指数大写
}
