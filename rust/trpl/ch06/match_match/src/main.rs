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

// 比如我们想要编写一个函数，它获取一个 Option<i32> ，如果其中含有一个值，将其加一。如果其中没有值，函数应该返回 None 值，而不尝试执行任何操作。

fn plus_one(x: Option<i32>) -> Option<i32> {
    match x {
        None => None,
        Some(i) => Some(i + 1),
    }
}

fn main() {
    println!("Coin::Penny {}", value_in_cents(Coin::Penny));
    println!("Coin::Nickel {}", value_in_cents(Coin::Nickel));
    println!("Coin::Dime {}", value_in_cents(Coin::Dime));
    println!(
        "Coin::Quarter {}",
        value_in_cents(Coin::Quarter(UsState::Alabama))
    );

    // plus one
    let five = Some(5);
    let six = plus_one(five);
    let none = plus_one(None);
    println!("{:?}", six);
    println!("{:?}", none);

    let dice_roll = 9;
    match dice_roll {
        3 => {
            println!("add_fancy_hat");
        }
        7 => {
            println!("remove_fancy_hat");
        }
        _ => {
            println!("move_player",)
        }
    }
}
