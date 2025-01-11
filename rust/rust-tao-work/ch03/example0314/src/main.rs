fn main() {
    let x = "1";
    // 可以指定类型
    println!("{:?}", x.parse::<i32>().unwrap());
}
