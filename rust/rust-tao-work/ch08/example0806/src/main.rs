fn main() {
    let mut a = String::from("foo");
    println!("{:p}", a.as_ptr()); // 堆中字节序列的地址
    println!("{:p}", &a); // 字符串变量在栈上中指针的地址
    assert_eq!(a.len(), 3);
    println!("{:?}", a.capacity());
    a.reserve(10);
    println!("{:?}", a.capacity());
}
