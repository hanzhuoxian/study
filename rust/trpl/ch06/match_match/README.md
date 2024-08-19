## match 控制流结构

```rust
enum Coin {
    Penny,
    Nickel,
    Dime,
    Quarter,
}

fn value_in_cents(coin: Coin) -> u8 {
    match coin {
        Coin::Penny => {
            1
        }
        Coin::Nickel => 5,
        Coin::Dime => 10,
        Coin::Quarter => 25,
    }
}
```

## 绑定值的模式

```rust
enum Coin {
    Penny,
    Nickel,
    Dime,
    Quarter(UsState),
}

#[derive(Debug)]
enum UsState {
    Alabama,
    Alaska,
}

fn value_in_cents(coin: Coin) -> u8 {
    match coin {
        Coin::Penny => {
            println!("luck penny!");
            1
        }
        Coin::Nickel => 5,
        Coin::Dime => 10,
        Coin::Quarter(state) => {
            println!("State quarter from {state:?}!");
            25
        }
    }
}
```

## 匹配 Option<T>

```rust

fn plus_one(x: Option<i32>) -> Option<i32> {
    match x {
        None => None,
        Some(i) => Some(i + 1),
    }
}

// 匹配 Some(T)
// plus one
let five = Some(5);
let six = plus_one(five); // 匹配 Some(T)
let none = plus_one(None); // 匹配 None
println!("{:?}", six);
println!("{:?}", none);
```

## match 枚举值必须是穷尽的

```rust
    // 下面的代码将报错，因为没有匹配 None
    match x {
        Some(i) => Some(i + 1),
    }
```

## 通配模式和 _ 占位符

```rust
    let dice_roll = 9;
    match dice_roll {
        3 => {
            println!("add_fancy_hat");
        }
        7 => {
            println!("remove_fancy_hat");
        }
        other => {
            println!("{} move_player", other)
        }
    }

    
    let dice_roll = 9;
    match dice_roll {
        3 => {
            println!("add_fancy_hat");
        }
        7 => {
            println!("remove_fancy_hat");
        }
        _ => {
            println!(" move_player")
        }
    }
```


