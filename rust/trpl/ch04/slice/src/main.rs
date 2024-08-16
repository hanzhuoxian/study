fn main() {
    let mut s = String::from("hello world!");
    let world = first_world(&s); // world 的值为5
    println!("{}", world);
    s.clear(); // 清空了字符串
               // world 在此处的值仍然是 5 但是没有更多的字符串可以应用这个值，world 变成一个无效的值

    // 字符串slice
    let s = String::from("hello world");
    let hello: &str = &s[0..5];
    let world: &str = &s[6..11];
    println!("{} {}", hello, world);

    let hello: &str = &s[..5]; // 省略开始索引
    let world: &str = &s[6..]; // 省略结束索引
    let hello_world = &s[..]; // 开始和结束索引都省略
    println!("{} {} {}", hello, world, hello_world);

    let mut s = String::from("hello world");
    let first = first_world_string(&s);
    // s.clear(); // 错误
    println!("slice first world {}", first);

    println!("first_world_str {}", first_world_str("hello world"));
    println!("first_world_str {}", first_world_str(&s));
    println!("first_world_str {}", first_world_str(&s[..6]));

    // 其他类型的 slice
    let a = [1, 2, 3, 4, 5];
    let slice = &a[1..3];
    assert_eq!(slice, &[2, 3]);
}

fn first_world(s: &String) -> usize {
    let bytes = s.as_bytes();
    for (i, &item) in bytes.iter().enumerate() {
        if item == b' ' {
            return i;
        }
    }
    s.len()
}

fn first_world_string(s: &String) -> &str {
    let bytes = s.as_bytes();
    let mut last_index = bytes.len();
    for (i, &item) in bytes.iter().enumerate() {
        if item == b' ' {
            last_index = i
        }
    }
    &s[..last_index]
}

fn first_world_str(s: &str) -> &str {
    let bytes = s.as_bytes();
    let mut last_index = bytes.len();
    for (i, &item) in bytes.iter().enumerate() {
        if item == b' ' {
            last_index = i
        }
    }
    &s[..last_index]
}
