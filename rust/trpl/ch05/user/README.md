## 创建结构体

```rust
let mut user1 = User {
    active: true,
    username: String::from("li"),
    email: String::from("li"),
    sign_in_count: 1,
};

// 使用结构体
println!("{}", user1.username);
//  结构体字段赋值
user1.active = false; // 结构体字段赋值
println!("{}", user1.active);
```

## 初始化简写语法

```rust

fn build_user(username: String, email: String) -> User {
    User {
        active: true,
        username: username,
        email, // 字段初始化简写语法 field init shorthand
        sign_in_count: 1,
    }
}

```


## 结构体更新语法

```rust
let user1 = build_user(String::from("LISI"), String::from("lisi@rust.com"));
// 结构体更新语法
let user2 = User {
    email: String::from("another@example.com"),
    ..user1 // ..user1 必须放在最后，以指定其余的字段应从 user1 的相应字段中获取其值
};
```

## 使用没有命名字段的元组结构体来创建不同的类型

```rust
    // 元组结构体
    struct Color(i32, i32, i32);
    struct Point(i32, i32, i32);
    let black = Color(0, 0, 0);
    let orign = Point(0, 0, 0);
```

## 类单元结构体（unit-like structs）

```rust

struct AlwaysEqual();
let subject = AlwaysEqual();
println!("{:?}", subject);
```

## 结构体所有权

```rust
struct User {
    active: bool,
    username: &str,
    email: &str,
    sign_in_count: u64,
}

fn main() {
    let user1 = User {
        active: true,
        username: "someusername123",
        email: "someone@example.com",
        sign_in_count: 1,
    };
}
```
