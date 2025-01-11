fn main() {
    let x = "1";
    // parse 是一个泛型方法，不知道应该转化成什么类型，需要明确标注类型
    println!("{:?}", x.parse().unwrap());
    // 可以指定类型
    println!("{:?}", x.parse::<i32>().unwrap());
}
