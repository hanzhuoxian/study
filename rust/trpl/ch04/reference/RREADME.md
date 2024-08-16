## 引用

使用 & 符号使用变量的引用而不获取所有权

```rust
let s1 = String::from("hello");

let len = calculate_length(&s1);
```
&s1 语法让我们创建一个指向 s1 的引用，但是并不拥有它。因为并不拥有这个值，所以当引用停止使用时，它所指向的值也不会被丢弃。

同理参数也使用 & 来表明参数 s 的类型是一个引用。
```rust
fn calculate_length(s: &String) -> usize { // s 是 String 的引用
    s.len()
} // 这里，s 离开了作用域。但因为它并不拥有引用值的所有权，
  // 所以什么也不会发生
```

我们将创建引用的行为成为借用（borrowing）

可变引用 &mut , 如果你有一个对该变量的可变引用那么就不能再创建对该变量的引用，同一时间只能对一个变量持有可变引用。可以避免数据竞争。


这个代码是编译报错的，w1 与 w2 想同时持有对变量 s1 的引用。
```rust
let mut s1 = String::from("rust");
let w1 = &mut s1;
// let w2 = &mut s1; // rustc: cannot borrow `s1` as mutable more than once at a time second mutable borrow occurs here
println!("w1 {w1}");
```

如下代码是可以正常运行的， 因为 w1 不再持有 s1 的引用时 w2 才创建对 s1 的引用。
```rust
    let mut s1 = String::from("rust mut");
    let w1 = &mut s1;
    println!("{w1}");
    let w2 = &mut s1;
    w2.push_str(" w2"); // no problem
    println!("{w2}")
```

一个引用的作用域从声明的地方开始一直持续到最后一次使用为止。w1 和 w2 的作用域没有重叠，所以是可以编译的。

## 悬垂引用 （Dangling References）

在具有指针的语言中，很容易通过释放内存时保留指向它的指针而错误的生成一个悬垂指针（Dangling Pointer），所谓悬垂指针是指其指向的内存
可能已经被分配给其他持有者。rust 中编译器会确保引用永远也不会变为悬垂状态，当你拥有一些数据的引用，
编译器确保数据不会在其引用之前离开作用域。

```rust
fn dangle() -> &String { // dangle 返回一个字符串的引用

    let s = String::from("hello"); // s 是一个新字符串

    &s // 返回字符串 s 的引用
} // 这里 s 离开作用域并被丢弃。其内存被释放。
  // 危险！
```

下面代码所有权被移动，所以没问题
```rust
fn no_dangle() -> String {
    let s = String::from("hello");

    s // 所有权被移动
}
```
