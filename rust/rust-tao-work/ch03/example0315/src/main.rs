fn main() {
    let a: i32 = 0;
    let a_pos = a.is_positive(); // `{integer}` 并非真实类型，只能推导出数字
    println!("{:?}", a_pos);
}
