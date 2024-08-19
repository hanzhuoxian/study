## 枚举定义

```rust
enum IpAddrKind {
    V4,
    V6,
}
```

## 枚举值

```rust
    let four = IpAddrKind::V4;
    let six: IpAddrKind = IpAddrKind::V6;

    fn route(ip_kind: IpAddrKind) {}

    route(IpAddrKind::V4);
    route(IpAddrKind::V6);

```

## 使用结构体

```rust

    struct IpAddrStruct {
        kind: IpAddrKind,
        address: String,
    }


    let home = IpAddrStruct {
        kind: IpAddrKind::V4,
        address: String::from("127.0.0.1"),
    };

    let loopback = IpAddrStruct {
        kind: IpAddrKind::V6,
        address: String::from("::1"),
    };
```

每一个我们定义的枚举成员的名字也变成了一个构建枚举的实例的函数。也就是说，IpAddr::V4() 是一个获取 String 参数并返回 IpAddr 类型实例的函数调用。作为定义枚举的结果，这些构造函数会自动被定义。

```rust
    enum IpAddr {
        V4(String),
        V6(String),
    }
    let home = IpAddr::V4(String::from("127.0.0.1"));
    let loopback = IpAddr::V6(String::from("::1"));

    // 每个成员都可以处理不同类型和数量的数据
    enum IpAddrU8 {
        V4(u8, u8, u8, u8),
        V6(String),
    }
    let home = IpAddrU8::V4(127, 0, 0, 1);
    let loopback = IpAddrU8::V6(String::from("::1"));

```

## 枚举方法

```rust
    enum Message {
        Quit,                       // 没有包含任何数据
        Move { x: i32, y: i32 },    // 类似结构体包含命名字段
        Write(String),              //包含单独一个String
        ChangeColor(i32, i32, i32), //包含三个 i32
    }

    impl Message {
        fn call(&self) {
        }
    }

    let message = Message::Write(String::from("hello"));
    message.call();
```

## Option 枚举

```rust
// option 定义
pub enum Option<T> {
    /// No value.
    #[lang = "None"]
    #[stable(feature = "rust1", since = "1.0.0")]
    None,
    /// Some value of type `T`.
    #[lang = "Some"]
    #[stable(feature = "rust1", since = "1.0.0")]
    Some(#[stable(feature = "rust1", since = "1.0.0")] T),
}

    // 成员使用

    let some_number = Option::Some(3);
    let some_char = Option::Some('e');
    let absent_number: Option<i32> = Option::None;
    
    let some_number = Some(3);
    let some_char = Some('e');
    let absent_number: Option<i32> = None;
```