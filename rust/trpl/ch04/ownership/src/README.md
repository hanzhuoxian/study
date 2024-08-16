## 所有权规则

1. Rust 中每一个值都有一个所有者（owner）。
2. 值在任一时刻有且只有一个所有者。
3. 当所有者离开作用域时，这个值将被丢弃。

## 变量作用域

```rust
{                      // s 在这里无效，它尚未声明
    let s = "hello";   // 从此处起，s 是有效的

    // 使用 s
}                      // 此作用域已结束，s 不再有效

```

## String 类型

```rust
    let mut s = String::from("hello");
    s.push_str(", world!");
    println!("String s : {s}");
```

就字符串字面值来说，我们在编译时就知道其内容，所以文本被直接编码进最终的可执行文件中，这使得字符串字面值快速且高效。
String 类型为了支持一个可变、可增长的文本片段，它需要在堆上分配一块内存来存放内容，这意味着：

- 必须在运行时向内存分配器（memory alloctor）请求内存。
- 需要一个当我们处理完 String 时将内存返回给分配器的方法。

第一部分由我们完成，当我们调用 `String::from` 时，它的实现请求其所需的内存。
内存在拥有它的变量离开作用域后就会被自动释放。当 `s` 离开作用域 `rust` 为我们调用一个特殊的函数 `drop`，
这里 String 的作者可以放置释放内存的代码。

## 变量与数据交互的方式：移动

```rust
let s1 = String::from("hello");
let s2 = s1;
println!("s2 = {s2}"); // s2 是有效的
                       // println!("s1 = {s1}, s2 = {s2}"); // 编译错误，s1 已经被移动到 s2
```

Rust 有一个叫做 Copy trait 的特殊注解，可以用在类似整型这样的存储在栈上的类型上（第十章将会详细讲解 trait）。
如果一个类型实现了 Copy trait，那么一个旧的变量在将其赋值给其他变量后仍然可用。

一些 copy 的类型:

- 所有整数类型，比如 u32。
- 布尔类型，bool，它的值是 true 和 false。
- 所有浮点数类型，比如 f64。
- 字符类型，char。
- 元组，当且仅当其包含的类型也都是 Copy 的时候。比如，(i32, i32) 是 Copy 的，但 (i32, String) 就不是。

## 变量与数据的交互方式二：克隆

```rust
// 变量与数据的交互方式二：克隆
let s1 = String::from("hello");
let s2 = s1.clone();
println!("s1 = {s1}, s2 = {s2}");
```

## 所有权与函数

```rust

fn main(){
    let s = String::from("hello"); // s 进入作用域
    takes_ownership(s); // s 的值移动到函数里 ...
                        // ... 所以到这里不再有效
                        // println!("{s}"); // 编译错误
    let x = 5; // x 进入作用域
    makes_copy(x); // x 应该移动函数里，
                // 但 i32 是 Copy 的，
                // 所以在后面可继续使用 x
    println!("x = {x}");
}

fn takes_ownership(some_string: String) {
    // some_string 进入作用域
    println!("{}", some_string);
} // 这里，some_string 移出作用域并调用 `drop` 方法。
  // 占用的内存被释放

fn makes_copy(some_integer: i32) {
    // some_integer 进入作用域
    println!("{}", some_integer);
} // 这里，some_integer 移出作用域。没有特殊之处
```

## 所有权规则
将值赋给另一个变量时移动它。当持有堆中数据值的变量离开作用域时，其值将通过 `drop` 被清理掉，除非数据被移动微另一个变量所有。
