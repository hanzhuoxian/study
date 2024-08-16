## 字符串 slice

字符串 slice （string slice）是 String 中一部分值的引用，它的类型写作 &str，它看起来像这样：
```rust
let s = String::from("hello world");
let hello: &str = &s[0..5];
let world: &str = &s[6..11];
println!("{} {}", hello, world);
```

不同于整个 String 的引用，hello 是一个部分 String 的引用，由一个额外的 [0..5] 部分指定。
可以使用一个由中括号中的 [starting_index..ending_index] 指定的 range 创建一个 slice，
其中 starting_index 是 slice 的第一个位置，ending_index 则是 slice 最后一个位置的后一个值。
在其内部，slice 的数据结构存储了 slice 的开始位置和长度，长度对应于 ending_index 减去 starting_index 的值。
所以对于 let world = &s[6..11]; 的情况，world 将是一个包含指向 s 索引 6 的指针和长度值 5 的 slice。

[string_index..end_index]， string_index 如果为 0 可以省略，end_index 如果为为最后一个字节，也可以舍弃尾部的数字。

```rust
    let hello = &s[..5]; // 省略开始索引
    let word = &s[6..]; // 省略结束索引
    let hello_word = &s[..]; // 开始和结束索引都省略
    println!("{} {} {}", hello, word, hello_word);
```

## 字符串字面值就是字符串 slice

```rust
let s = "Hello, world!";
```

这里 s 的类型是 &str：它是一个指向二进制程序特定位置的 slice。这也就是为什么字符串字面值是不可变的；&str 是一个不可变引用。
