fn main() {
    let str = "Hello Rust"; // 是胖指针（Fat Pointer）
    let ptr = str.as_ptr(); // 获取字符串字面量的存储地址
    let len = str.len(); // 获长度符串长度
    println!("{:p}", ptr);
    println!("{:?}", len);
}
