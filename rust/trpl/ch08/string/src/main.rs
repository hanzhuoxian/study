fn main() {
    // 新建字符串
    let mut s = String::new();
    s.push_str("hello");
    s.push(' ');
    println!("{s}");

    let data = "initial contents";
    let s = data.to_string();
    println!("{s} {}", "initial contents".to_string());

    // "initial contents".to_string() 等同于 String::from("initial contents")
    let s = String::from("initial contents");

    let mut s1 = String::from("foo");
    let s2 = "bar"; // s2 未被移动
    s1.push_str(s2);
    println!("s2 is {s2}");

    // 使用 + 符号拼接字符串
    let s1 = String::from("Hello, ");
    let s2 = String::from("world!");
    let s3 = s1 + &s2; // 注意 s1 被移动了，不能继续使用
    println!("{s2} {s3}");

    // 使用 format! 宏拼接字符串
    let s1 = String::from("tic");
    let s2 = String::from("tac");
    let s3 = String::from("toe");
    let s = format!("{s1}-{s2}-{s3}");
    println!("{}", s);

    // 索引字符串
    // s[0] 会报错，因为 Rust 不支持索引字符串

    let hello = "hello, 世界";

    for c in hello.chars() {
        // 明确返回字符
        println!("{}", c);
    }

    for b in hello.bytes() {
        // 明确返回字节
        println!("{}", b);
    }
}
